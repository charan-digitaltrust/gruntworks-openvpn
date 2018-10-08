package test

import (
	"fmt"
	"github.com/gruntwork-io/terratest/modules/aws"
	"github.com/gruntwork-io/terratest/modules/git"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/packer"
	"github.com/gruntwork-io/terratest/modules/retry"
	"github.com/gruntwork-io/terratest/modules/ssh"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"strconv"
	"strings"
	"testing"
	"time"
)

type suite struct {
	terraformOptions   *terraform.Options
	keyPair            *aws.Ec2Keypair
	accountId          string
	region             string
	ipAddress          string
	host               ssh.Host
	output             string
	instanceId         string
}

func TestOpenVpnInitializationSuite(t *testing.T) {
	testSuite := suite{
		ipAddress:"",
	}

	testSuite.region = aws.GetRandomRegion(t, []string{"ca-central-1"}, nil)
	testSuite.accountId = aws.GetAccountId(t)

	//Build the AMI with packer
	packerOptions := &packer.Options{
		Template: "../examples/packer/openvpn-server-ubuntu1604.json",
		Vars: map[string]string{
			"aws_region": testSuite.region,
			"aws_account_id": testSuite.accountId,
			"package_openvpn_branch": git.GetCurrentBranchName(t),
			"active_git_branch": git.GetCurrentBranchName(t),
		},
	}

	amiId := packer.BuildArtifact(t, packerOptions)

	vpc := aws.GetDefaultVpc(t, testSuite.region)

	if len(vpc.Subnets) == 0 {
		t.Fatalf("Default vpc %s contained no subnets", vpc.Id)
	}

	testSuite.keyPair = aws.CreateAndImportEC2KeyPair(t, testSuite.region, getRandomizedString("tst-openvpn-key"))

	//Setup terratest
	testSuite.terraformOptions = &terraform.Options{
		TerraformDir: "../examples/openvpn-host",
		Vars: map[string]interface{}{
			"name": getRandomizedString("tst-openvpn-host"),
			"backup_bucket_name": getRandomizedString("tst-openvpn"),
			"request_queue_name": getRandomizedString("tst-openvpn-requests"),
			"revocation_queue_name": getRandomizedString("tst-openvpn-revocations"),
			"ami": amiId,
			"keypair_name": testSuite.keyPair.Name,
			"aws_account_id": testSuite.accountId,
			"aws_region": testSuite.region,
		},
	}


	terraform.InitAndApply(t, testSuite.terraformOptions)

	defer func() {
		terraform.Destroy(t, testSuite.terraformOptions)

		aws.DeleteAmi(t, testSuite.region, amiId)
		logger.Log(t, "TearDownSuite Complete")
	}()

	testSuite.ipAddress = terraform.Output(t, testSuite.terraformOptions, "openvpn_host_public_ip")

	waitUntilSshAvailable(t, &testSuite)
	waitUntilOpenVpnInitComplete(t, &testSuite)

	t.Log("SetupSuite Complete, Running Tests")

	t.Run("openvpn tests", func(t *testing.T) {
		t.Run("running testOpenVpnIsRunning", wrapTestCase(testOpenVpnIsRunning, &testSuite))
		t.Run("running testOpenVpnAdminProcessRequestsIsRunning", wrapTestCase(testOpenVpnAdminProcessRequestsIsRunning, &testSuite))
		t.Run("running testOpenVpnAdminProcessRevokesIsRunning", wrapTestCase(testOpenVpnAdminProcessRevokesIsRunning, &testSuite))
		t.Run("running testCrlExpirationDateUpdated", wrapTestCase(testCrlExpirationDateUpdated, &testSuite))
		t.Run("running testCronJobExists", wrapTestCase(testCronJobExists, &testSuite))
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
		SshKeyPair: testSuite.keyPair.KeyPair,
	}

	retry.DoWithRetry(
		t,
		fmt.Sprintf("SSH to public host %s", testSuite.ipAddress),
		20,
		30 * time.Second,
		func() (string, error) {
			return "", ssh.CheckSshConnectionE(t, testSuite.host)
		},
	)

}

func waitUntilOpenVpnInitComplete(t *testing.T, testSuite *suite) {
	retry.DoWithRetry(
		t,
		fmt.Sprintf("Waiting for OpenVPN initialization to complete"),
		75,
		30 * time.Second,
		func() (string, error) {
			return "", initComplete(t, testSuite)
		},
	)
}

func initComplete(t *testing.T, testSuite *suite) (error) {
	command := "sudo ls /etc/openvpn/openvpn-init-complete"
	output, err := ssh.CheckSshCommandE(t, testSuite.host, command)

	if strings.Contains(output, "such file or directory") {
		return fmt.Errorf("OpenVPN initialization not yet complete")
	}

	if err != nil {
		return err
	}

	return nil
}

func testOpenVpnIsRunning(t *testing.T, testSuite *suite) {
	commandToTest := "sudo ps -ef|grep openvpn"
	testSuite.output = ssh.CheckSshCommand(t, testSuite.host, commandToTest)

	// It will be convenient to see the full command output directly in logs. This will show only when there's a test failure.
	t.Logf("Result of running \"%s\"\n", commandToTest)
	t.Log(testSuite.output)

	assert.Contains(t, testSuite.output, "/usr/sbin/openvpn")
}

func testOpenVpnAdminProcessRequestsIsRunning(t *testing.T, testSuite *suite) {
	commandToTest := "sudo ps -ef|grep openvpn"
	testSuite.output = ssh.CheckSshCommand(t, testSuite.host, commandToTest)

	// It will be convenient to see the full command output directly in logs. This will show only when there's a test failure.
	t.Logf("Result of running \"%s\"\n", commandToTest)
	t.Log(testSuite.output)

	assert.Contains(t, testSuite.output, "/usr/local/bin/openvpn-admin process-requests")
}

func testOpenVpnAdminProcessRevokesIsRunning(t *testing.T, testSuite *suite) {
	commandToTest := "sudo ps -ef|grep openvpn"
	var err error
	testSuite.output = ssh.CheckSshCommand(t, testSuite.host, commandToTest)
	if err != nil {
		t.Fatalf("Failed to SSH to AMI Builder at %s and execute command :%s\n", testSuite.ipAddress, err.Error())
	}

	// It will be convenient to see the full command output directly in logs. This will show only when there's a test failure.
	t.Logf("Result of running \"%s\"\n", commandToTest)
	t.Log(testSuite.output)

	assert.Contains(t, testSuite.output, "/usr/local/bin/openvpn-admin process-revokes")
}

func testCronJobExists(t *testing.T, testSuite *suite) {
	commandToTest := "sudo cat /etc/cron.hourly/backup-openvpn-pki"
	testSuite.output = ssh.CheckSshCommand(t, testSuite.host, commandToTest)

	// It will be convenient to see the full command output directly in logs. This will show only when there's a test failure.
	t.Logf("Result of running \"%s\"\n", commandToTest)
	t.Log(testSuite.output)

	assert.Contains(t, testSuite.output, "backup-openvpn-pki")
}

func testCrlExpirationDateUpdated(t *testing.T, testSuite *suite) {
	commandToTest := "sudo cat /etc/openvpn-ca/openssl-1.0.0.cnf"
	testSuite.output = ssh.CheckSshCommand(t, testSuite.host, commandToTest)

	// It will be convenient to see the full command output directly in logs. This will show only when there's a test failure.
	t.Logf("Result of running \"%s\"\n", commandToTest)
	t.Log(testSuite.output)

	assert.Contains(t, testSuite.output, "default_crl_days= 3650")
}
