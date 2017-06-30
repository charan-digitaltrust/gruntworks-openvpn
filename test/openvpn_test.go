package test

import (
	"testing"
	terralog "github.com/gruntwork-io/terratest/log"
	"github.com/gruntwork-io/terratest/resources"
	"github.com/gruntwork-io/terratest"
	"github.com/gruntwork-io/terratest/packer"
	"fmt"
	"time"
	"github.com/gruntwork-io/terratest/git"
	"github.com/gruntwork-io/terratest/aws"
	"github.com/stretchr/testify/assert"
	"github.com/gruntwork-io/terratest/ssh"
	"strings"
	"github.com/gruntwork-io/terratest/util"
)

type openVpnSuite struct {
	terratest.TestSuiteBase
	ipAddress  string
	host       ssh.Host
	instanceId string
}

func TestOpenVpnInitializationSuite(t *testing.T) {
	testSuite := openVpnSuite{
		ipAddress:"",
	}

	testSuite.Logger = terralog.NewLogger("TestOpenVpnSuite")
	testSuite.ResourceCollection = resources.CreateBaseRandomResourceCollection(t, resources.REGIONS_WITHOUT_T2_NANO...)

	//Build the AMI with packer
	packerOptions := packer.PackerOptions{
		Template: "../examples/packer/openvpn-server-ubuntu1604.json",
		Vars: map[string]string{
			"aws_region": testSuite.ResourceCollection.AwsRegion,
			"package_openvpn_branch": git.GetCurrentBranchName(t),
			"active_git_branch": git.GetCurrentBranchName(t),
		},
	}

	amiId, err := packer.BuildAmi(packerOptions, testSuite.Logger)
	if err != nil || len(amiId) == 0 {
		t.Fatalf("Failed to build AMI due to error: %s", err.Error())
	}

	vpc, err := testSuite.ResourceCollection.GetDefaultVpc()
	if err != nil {
		t.Fatalf("Failed to get default VPC: %s\n", err.Error())
	}

	if len(vpc.Subnets) == 0 {
		t.Fatalf("Default vpc %s contained no subnets", vpc.Id)
	}

	//Setup terratest
	testSuite.TerratestOptions = resources.CreateBaseTerratestOptions(t, "OpenVpnTestTerraform", "../examples/openvpn-host", testSuite.ResourceCollection)

	kmsKey, err := aws.GetDedicatedTestKeyArn(testSuite.ResourceCollection.AwsRegion)
	if err != nil {
		t.Fatalf("Error while retreiving dedicated test KMS key - %s", err.Error())
	}
	testSuite.TerratestOptions.Vars["name"] = fmt.Sprintf("tst-openvpn-host", testSuite.ResourceCollection.UniqueId)
	testSuite.TerratestOptions.Vars["backup_bucket_name"] = fmt.Sprintf("tst-openvpn", testSuite.ResourceCollection.UniqueId)
	testSuite.TerratestOptions.Vars["request_queue_name"] = fmt.Sprintf("tst-openvpn-requests", testSuite.ResourceCollection.UniqueId)
	testSuite.TerratestOptions.Vars["revocation_queue_name"] = fmt.Sprintf("tst-openvpn-revocations", testSuite.ResourceCollection.UniqueId)
	testSuite.TerratestOptions.Vars["ami"] = amiId
	testSuite.TerratestOptions.Vars["keypair_name"] = testSuite.ResourceCollection.KeyPair.Name
	testSuite.TerratestOptions.Vars["backup_kms_key"] = kmsKey


	defer func() {
		ec2Client, err := aws.CreateEC2Client(testSuite.ResourceCollection.AwsRegion)
		if err != nil {
			t.Fatal("Error creating EC2 Client")
		}

		_, err = terratest.Destroy(testSuite.TerratestOptions, testSuite.ResourceCollection)
		if err != nil {
			t.Fatal("Terraform Destroy failed")
		}

		aws.RemoveAmi(ec2Client, amiId)
		testSuite.Logger.Println("TearDownSuite Complete")
	}()

	_, err = terratest.Apply(testSuite.TerratestOptions)
	if err != nil {
		t.Fatalf("Unexpected error when applying terraform templates: %v", err)
	}

	testSuite.ipAddress, err = terratest.Output(testSuite.TerratestOptions, "openvpn_host_public_ip")
	if err != nil {
		t.Fatalf("An error occurred retreiving terraform output - %s", err.Error())
	}

	waitUntilSshAvailable(t, &testSuite)
	waitUntilOpenVpnInitComplete(t, &testSuite)

	testSuite.Logger.Println("SetupSuite Complete")

	t.Run("openvpn tests", func(t *testing.T) {
		t.Run("running test", wrapOpenVpnTestCase(testOpenVpnIsRunning, &testSuite))
		t.Run("running test", wrapOpenVpnTestCase(testOpenVpnAdminProcessCertsIsRunning, &testSuite))
		t.Run("running test", wrapOpenVpnTestCase(testOpenVpnRestoresFromS3Correctly, &testSuite))
	})
}

func wrapOpenVpnTestCase(testCase func(t *testing.T, testSuite *openVpnSuite), testSuite *openVpnSuite) func(t *testing.T) {
	return func(t *testing.T) {
		testCase(t, testSuite)
	}
}


func waitUntilSshAvailable(t *testing.T, testSuite *openVpnSuite) {
	// SSH into EC2 Instance
	testSuite.host = ssh.Host{
		Hostname: testSuite.ipAddress,
		SshUserName: "ubuntu",
		SshKeyPair: testSuite.ResourceCollection.KeyPair,
	}

	_, err := util.DoWithRetry(
		fmt.Sprintf("SSH to public host %s", testSuite.ipAddress),
		10,
		30 * time.Second,
		testSuite.Logger,
		func() (string, error) {
			return "", ssh.CheckSshConnection(testSuite.host, testSuite.Logger)
		},
	)

	if err != nil {
		t.Fatalf("Failed to SSH to host at %s :%s\n", testSuite.ipAddress, err.Error())
	}
}

func waitUntilOpenVpnInitComplete(t *testing.T, testSuite *openVpnSuite) {
	_, err := util.DoWithRetry(
		fmt.Sprintf("Waiting for OpenVPN initialization to complete"),
		60,
		30 * time.Second,
		testSuite.Logger,
		func() (string, error) {
			return "", initComplete(t, testSuite)
		},
	)

	if err != nil {
		t.Fatal("OpenVPN initilization failed to complete")
	}
}

func initComplete(t *testing.T, testSuite *openVpnSuite) (error) {
	command := "sudo ls /etc/openvpn/openvpn-init-complete"
	output, err := ssh.CheckSshCommand(testSuite.host, command, testSuite.Logger)

	if strings.Contains(output, "no such file or directory") {
		return fmt.Errorf("initialization not yet complete")
	}

	if err != nil {
		return err
	}

	return nil
}

func getInstanceId(t *testing.T, testSuite *openVpnSuite) string {
	//Get the instance id of the running instance
	command := "curl --silent http://169.254.169.254/latest/meta-data/instance-id"
	output, err := ssh.CheckSshCommand(testSuite.host, command, testSuite.Logger)
	if err != nil {
		t.Fatalf("Unable to get instance id :\n", err.Error())
	}

	return output
}

func testOpenVpnIsRunning(t *testing.T, testSuite *openVpnSuite) {
	commandToTest := "sudo ps -ef|grep openvpn"
	var err error
	testSuite.TerraformOutput, err = ssh.CheckSshCommand(testSuite.host, commandToTest, testSuite.Logger)
	if err != nil {
		t.Fatalf("Failed to SSH to AMI Builder at %s and execute command :%s\n", testSuite.ipAddress, err.Error())
	}

	// It will be convenient to see the full command output directly in logs. This will show only when there's a test failure.
	t.Logf("Result of running \"%s\"\n", commandToTest)
	t.Log(testSuite.TerraformOutput)

	assert.Contains(t, testSuite.TerraformOutput, "/usr/sbin/openvpn")
}

func testOpenVpnAdminProcessCertsIsRunning(t *testing.T, testSuite *openVpnSuite) {
	commandToTest := "sudo ps -ef|grep openvpn"
	var err error
	testSuite.TerraformOutput, err = ssh.CheckSshCommand(testSuite.host, commandToTest, testSuite.Logger)
	if err != nil {
		t.Fatalf("Failed to SSH to AMI Builder at %s and execute command :%s\n", testSuite.ipAddress, err.Error())
	}

	// It will be convenient to see the full command output directly in logs. This will show only when there's a test failure.
	t.Logf("Result of running \"%s\"\n", commandToTest)
	t.Log(testSuite.TerraformOutput)

	assert.Contains(t, testSuite.TerraformOutput, "openvpn-admin process-requests")
}

func testOpenVpnRestoresFromS3Correctly(t *testing.T, testSuite *openVpnSuite) {
	//Terminate the instance
	instanceId := getInstanceId(t, testSuite)
	ec2Client, err := aws.CreateEC2Client(testSuite.ResourceCollection.AwsRegion)
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
		t.Run("running test", wrapOpenVpnTestCase(testOpenVpnIsRunning, testSuite))
		t.Run("running test", wrapOpenVpnTestCase(testOpenVpnAdminProcessCertsIsRunning, testSuite))
	})
}
