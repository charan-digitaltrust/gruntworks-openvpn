package test

import (
	"testing"
	terralog "github.com/gruntwork-io/terratest/log"
	"github.com/gruntwork-io/terratest/resources"
	"github.com/gruntwork-io/terratest"
	"github.com/gruntwork-io/terratest/packer"
	"fmt"
	"time"
	"strconv"
	"github.com/gruntwork-io/terratest/git"
	"github.com/gruntwork-io/terratest/aws"
	"github.com/stretchr/testify/assert"
	"github.com/gruntwork-io/terratest/ssh"
	"strings"
	"log"
	"github.com/gruntwork-io/terratest/util"
)

type suite struct {
	logger             *log.Logger
	resourceCollection *terratest.RandomResourceCollection
	terratestOptions   *terratest.TerratestOptions
	ipAddress          string
	host               ssh.Host
	output             string
	instanceId         string
}

func TestOpenVpnInitializationSuite(t *testing.T) {
	testSuite := suite{
		ipAddress:"",
	}

	testSuite.logger = terralog.NewLogger("TestOpenVpnSuite")
	testSuite.resourceCollection = resources.CreateBaseRandomResourceCollection(t, resources.REGIONS_WITHOUT_T2_NANO...)

	//Build the AMI with packer
	packerOptions := packer.PackerOptions{
		Template: "../examples/packer/openvpn-server-ubuntu1604.json",
		Vars: map[string]string{
			"aws_region": testSuite.resourceCollection.AwsRegion,
			"aws_account_id": testSuite.resourceCollection.AccountId,
			"package_openvpn_branch": git.GetCurrentBranchName(t),
			"active_git_branch": git.GetCurrentBranchName(t),
		},
	}

	amiId, err := packer.BuildAmi(packerOptions, testSuite.logger)
	if err != nil || len(amiId) == 0 {
		t.Fatalf("Failed to build AMI due to error: %s", err.Error())
	}

	vpc, err := testSuite.resourceCollection.GetDefaultVpc()
	if err != nil {
		t.Fatalf("Failed to get default VPC: %s\n", err.Error())
	}

	if len(vpc.Subnets) == 0 {
		t.Fatalf("Default vpc %s contained no subnets", vpc.Id)
	}

	//Setup terratest
	testSuite.terratestOptions = resources.CreateBaseTerratestOptions(t, "OpenVpnTestTerraform", "../examples/openvpn-host", testSuite.resourceCollection)

	testSuite.terratestOptions.Vars["name"] = getRandomizedString("tst-openvpn-host")
	testSuite.terratestOptions.Vars["backup_bucket_name"] = getRandomizedString("tst-openvpn")
	testSuite.terratestOptions.Vars["request_queue_name"] = getRandomizedString("tst-openvpn-requests")
	testSuite.terratestOptions.Vars["revocation_queue_name"] = getRandomizedString("tst-openvpn-revocations")
	testSuite.terratestOptions.Vars["ami"] = amiId
	testSuite.terratestOptions.Vars["keypair_name"] = testSuite.resourceCollection.KeyPair.Name

	_, err = terratest.Apply(testSuite.terratestOptions)
	if err != nil {
		t.Fatalf("Unexpected error when applying terraform templates: %v", err)
	}

	defer func() {
		ec2Client, err := aws.CreateEC2Client(testSuite.resourceCollection.AwsRegion)
		if err != nil {
			t.Fatal("Error creating EC2 Client")
		}

		_, err = terratest.Destroy(testSuite.terratestOptions, testSuite.resourceCollection)
		if err != nil {
			t.Fatal("Terraform Destroy failed")
		}

		aws.RemoveAmi(ec2Client, amiId)
		testSuite.logger.Println("TearDownSuite Complete")
	}()

	testSuite.ipAddress, err = terratest.Output(testSuite.terratestOptions, "openvpn_host_public_ip")
	if err != nil {
		t.Fatalf("An error occurred retreiving terraform output - %s", err.Error())
	}

	waitUntilSshAvailable(t, &testSuite)
	waitUntilOpenVpnInitComplete(t, &testSuite)

	testSuite.logger.Println("SetupSuite Complete")

	t.Run("openvpn tests", func(t *testing.T) {
		t.Run("running test", wrapTestCase(testOpenVpnIsRunning, &testSuite))
		t.Run("running test", wrapTestCase(testOpenVpnAdminProcessCertsIsRunning, &testSuite))
		t.Run("running test", wrapTestCase(testOpenVpnRestoresFromS3Correctly, &testSuite))
		t.Run("running test", wrapTestCase(testCronJobExists, &testSuite))
	})
}

func wrapTestCase(testCase func(t *testing.T, testSuite *suite), testSuite *suite) func(t *testing.T) {
	return func(t *testing.T) {
		testCase(t, testSuite)
	}
}

func getRandomizedString(prefix string) (string) {
	return fmt.Sprintf("%s-%s", prefix, strconv.FormatInt(time.Now().Unix(), 10))
}

func waitUntilSshAvailable(t *testing.T, testSuite *suite) {
	// SSH into EC2 Instance
	testSuite.host = ssh.Host{
		Hostname: testSuite.ipAddress,
		SshUserName: "ubuntu",
		SshKeyPair: testSuite.resourceCollection.KeyPair,
	}

	_, err := util.DoWithRetry(
		fmt.Sprintf("SSH to public host %s", testSuite.ipAddress),
		10,
		30 * time.Second,
		testSuite.logger,
		func() (string, error) {
			return "", ssh.CheckSshConnection(testSuite.host, testSuite.logger)
		},
	)

	if err != nil {
		t.Fatalf("Failed to SSH to host at %s :%s\n", testSuite.ipAddress, err.Error())
	}
}

func waitUntilOpenVpnInitComplete(t *testing.T, testSuite *suite) {
	_, err := util.DoWithRetry(
		fmt.Sprintf("Waiting for OpenVPN initialization to complete"),
		60,
		30 * time.Second,
		testSuite.logger,
		func() (string, error) {
			return "", initComplete(t, testSuite)
		},
	)

	if err != nil {
		t.Fatal("OpenVPN initilization failed to complete")
	}
}

func initComplete(t *testing.T, testSuite *suite) (error) {
	command := "sudo ls /etc/openvpn/openvpn-init-complete"
	output, err := ssh.CheckSshCommand(testSuite.host, command, testSuite.logger)

	if strings.Contains(output, "no such file or directory") {
		return fmt.Errorf("initialization not yet complete")
	}

	if err != nil {
		return err
	}

	return nil
}

func getInstanceId(t *testing.T, testSuite *suite) string {
	//Get the instance id of the running instance
	command := "curl --silent http://169.254.169.254/latest/meta-data/instance-id"
	output, err := ssh.CheckSshCommand(testSuite.host, command, testSuite.logger)
	if err != nil {
		t.Fatalf("Unable to get instance id :\n", err.Error())
	}

	return output
}

func testOpenVpnIsRunning(t *testing.T, testSuite *suite) {
	commandToTest := "sudo ps -ef|grep openvpn"
	var err error
	testSuite.output, err = ssh.CheckSshCommand(testSuite.host, commandToTest, testSuite.logger)
	if err != nil {
		t.Fatalf("Failed to SSH to AMI Builder at %s and execute command :%s\n", testSuite.ipAddress, err.Error())
	}

	// It will be convenient to see the full command output directly in logs. This will show only when there's a test failure.
	t.Logf("Result of running \"%s\"\n", commandToTest)
	t.Log(testSuite.output)

	assert.Contains(t, testSuite.output, "/usr/sbin/openvpn")
}

func testOpenVpnAdminProcessCertsIsRunning(t *testing.T, testSuite *suite) {
	commandToTest := "sudo ps -ef|grep openvpn"
	var err error
	testSuite.output, err = ssh.CheckSshCommand(testSuite.host, commandToTest, testSuite.logger)
	if err != nil {
		t.Fatalf("Failed to SSH to AMI Builder at %s and execute command :%s\n", testSuite.ipAddress, err.Error())
	}

	// It will be convenient to see the full command output directly in logs. This will show only when there's a test failure.
	t.Logf("Result of running \"%s\"\n", commandToTest)
	t.Log(testSuite.output)

	assert.Contains(t, testSuite.output, "openvpn-admin process-requests")
}

func testOpenVpnRestoresFromS3Correctly(t *testing.T, testSuite *suite) {
	//Terminate the instance
	instanceId := getInstanceId(t, testSuite)
	ec2Client, err := aws.CreateEC2Client(testSuite.resourceCollection.AwsRegion)
	if err != nil {
		t.Fatalf("An error occurred creating ec2client\n", err)
	}

	err = aws.TerminateInstance(ec2Client, instanceId)
	if err != nil {
		t.Fatalf("An error occurred while terminating instance %s\n", instanceId, err)
	}

	waitUntilSshAvailable(t, testSuite)
	waitUntilOpenVpnInitComplete(t, testSuite)

	t.Run("openvpn restore tests", func(t *testing.T) {
		t.Run("running test", wrapTestCase(testOpenVpnIsRunning, testSuite))
		t.Run("running test", wrapTestCase(testOpenVpnAdminProcessCertsIsRunning, testSuite))
	})
}

func testCronJobExists(t *testing.T, testSuite *suite) {
	commandToTest := "sudo cat /etc/cron.hourly/backup-openvpn-pki"
	var err error
	testSuite.output, err = ssh.CheckSshCommand(testSuite.host, commandToTest, testSuite.logger)
	if err != nil {
		t.Fatalf("Failed to SSH to AMI Builder at %s and execute command :%s\n", testSuite.ipAddress, err.Error())
	}

	// It will be convenient to see the full command output directly in logs. This will show only when there's a test failure.
	t.Logf("Result of running \"%s\"\n", commandToTest)
	t.Log(testSuite.output)

	assert.Contains(t, testSuite.output, "backup-openvpn-pki")
}
