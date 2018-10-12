package test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/aws"
	"github.com/gruntwork-io/terratest/modules/git"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/packer"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/retry"
	"github.com/gruntwork-io/terratest/modules/ssh"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/gruntwork-io/terratest/modules/test-structure"
	"github.com/stretchr/testify/assert"
)

func TestOpenVpnInitializationUbuntuXenial(t *testing.T) {
	t.Parallel()
	testOpenVpnInitializationSuite(t, "ubuntu-16")
}

func testOpenVpnInitializationSuite(t *testing.T, osName string) {
	workingDir := test_structure.CopyTerraformFolderToTemp(t, "../", "examples/openvpn-host")

	// At the end of the test, delete the AMI
	defer test_structure.RunTestStage(t, "cleanup_ami", func() {
		awsRegion := test_structure.LoadString(t, workingDir, "awsRegion")
		deleteAMI(t, awsRegion, workingDir)
	})

	// At the end of the test, undeploy the web app using Terraform
	defer test_structure.RunTestStage(t, "cleanup_terraform", func() {
		undeployUsingTerraform(t, workingDir)
	})

	// At the end of the test, fetch the most recent syslog entries from each Instance. This can be useful for
	// debugging issues without having to manually SSH to the server.
	defer test_structure.RunTestStage(t, "logs", func() {
		if t.Failed() {
			logger.Log(t, "Fetching logs to help debug test failure.")
			awsRegion := test_structure.LoadString(t, workingDir, "awsRegion")
			fetchSyslogForInstance(t, osName, awsRegion, workingDir)
		}
	})

	// Build the AMI for the web app
	test_structure.RunTestStage(t, "build_ami", func() {
		// Pick a random AWS region to test in. This helps ensure your code works in all regions.
		awsRegion := aws.GetRandomRegion(t, nil, []string{"ap-northeast-1"})
		test_structure.SaveString(t, workingDir, "awsRegion", awsRegion)
		buildAMI(t, awsRegion, osName, workingDir)
	})

	// Create a test container using Terraform
	test_structure.RunTestStage(t, "deploy_terraform", func() {
		awsRegion := test_structure.LoadString(t, workingDir, "awsRegion")
		deployUsingTerraform(t, awsRegion, workingDir)
	})

	// Validate that the web app deployed and is responding to HTTP requests
	test_structure.RunTestStage(t, "validate", func() {
		validateInstanceRunningOpenVPNServer(t, osName, workingDir)
	})

}

// Build the AMI with packer
func buildAMI(t *testing.T, awsRegion string, osName string, workingDir string) {
	packerOptions := &packer.Options{
		// The path to where the Packer template is located
		Template: "../examples/packer/build.json",

		// Only build the AMI
		Only: fmt.Sprintf("%s-build", osName),

		// Variables to pass to our Packer build using -var options
		Vars: map[string]string{
			"aws_region":             awsRegion,
			"package_openvpn_branch": git.GetCurrentBranchName(t),
			"active_git_branch":      git.GetCurrentBranchName(t),
		},
	}

	// Save the Packer Options so future test stages can use them
	test_structure.SavePackerOptions(t, workingDir, packerOptions)

	// Build the AMI
	amiID := packer.BuildArtifact(t, packerOptions)

	// Save the AMI ID so future test stages can use them
	test_structure.SaveArtifactID(t, workingDir, amiID)
}

// Delete the AMI
func deleteAMI(t *testing.T, awsRegion string, workingDir string) {
	// Load the AMI ID and Packer Options saved by the earlier build_ami stage
	amiID := test_structure.LoadArtifactID(t, workingDir)

	aws.DeleteAmi(t, awsRegion, amiID)
}

// Deploy the terraform-packer-example using Terraform
func deployUsingTerraform(t *testing.T, awsRegion string, workingDir string) {
	// A unique ID we can use to namespace resources so we don't clash with anything already in the AWS account or
	// tests running in parallel
	uniqueID := random.UniqueId()

	// Give this EC2 Instance and other resources in the Terraform code a name with a unique ID so it doesn't clash
	// with anything else in the AWS account.
	instanceName := fmt.Sprintf("tst-openvpn-host-%s", uniqueID)

	// Load the AMI ID saved by the earlier build_ami stage
	amiID := test_structure.LoadArtifactID(t, workingDir)

	// Test that there are subnets in our vpc
	vpc := aws.GetDefaultVpc(t, awsRegion)
	if len(vpc.Subnets) == 0 {
		t.Fatalf("Default vpc %s contained no subnets", vpc.Id)
	}

	// Create a keypair to use to connect to the server
	keyPairName := fmt.Sprintf("tst-openvpn-key-%s", uniqueID)
	keyPair := aws.CreateAndImportEC2KeyPair(t, awsRegion, keyPairName)
	test_structure.SaveEc2KeyPair(t, workingDir, keyPair)

	terraformOptions := &terraform.Options{
		// The path to where our Terraform code is located
		TerraformDir: workingDir,

		// Variables to pass to our Terraform code using -var options
		Vars: map[string]interface{}{
			"aws_account_id":        aws.GetAccountId(t),
			"aws_region":            awsRegion,
			"name":                  instanceName,
			"ami_id":                amiID,
			"backup_bucket_name":    strings.ToLower(fmt.Sprintf("tst-openvpn-%s", uniqueID)),
			"request_queue_name":    fmt.Sprintf("tst-openvpn-requests-%s", uniqueID),
			"revocation_queue_name": fmt.Sprintf("tst-openvpn-revocations-%s", uniqueID),
			"keypair_name":          keyPair.Name,
		},
	}

	// Save the Terraform Options struct, instance name, and instance text so future test stages can use it
	test_structure.SaveTerraformOptions(t, workingDir, terraformOptions)

	// This will run `terraform init` and `terraform apply` and fail the test if there are any errors
	terraform.InitAndApply(t, terraformOptions)
}

// Undeploy the terraform-packer-example using Terraform
func undeployUsingTerraform(t *testing.T, workingDir string) {
	// Cleanup the keypair that we created earlier
	keyPair := test_structure.LoadEc2KeyPair(t, workingDir)
	aws.DeleteEC2KeyPair(t, keyPair)

	// Load the Terraform Options saved by the earlier deploy_terraform stage
	terraformOptions := test_structure.LoadTerraformOptions(t, workingDir)

	terraform.Destroy(t, terraformOptions)
}

// Fetch the most recent syslogs for the instance. This is a handy way to see what happened on the Instance as part of
// your test log output, without having to re-run the test and manually SSH to the Instance.
func fetchSyslogForInstance(t *testing.T, osName string, awsRegion string, workingDir string) {
	// Load the Terraform Options saved by the earlier deploy_terraform stage
	terraformOptions := test_structure.LoadTerraformOptions(t, workingDir)

	asgName := terraformOptions.Vars["name"].(string)

	sshUserName := osToSSHUserName(t, osName)

	keyPair := test_structure.LoadEc2KeyPair(t, workingDir)

	logFileSpec := aws.RemoteFileSpecification{
		AsgNames:               []string{asgName},
		RemotePathToFileFilter: osToLogPathSpec(t, osName),
		UseSudo:                false,
		SshUser:                sshUserName,
		KeyPair:                keyPair,
		LocalDestinationDir:    workingDir,
	}

	aws.FetchFilesFromAsgs(t, awsRegion, logFileSpec)

}

// Validate the openvpn server has been deployed and is working
func validateInstanceRunningOpenVPNServer(t *testing.T, osName string, workingDir string) {
	// Load the Terraform Options saved by the earlier deploy_terraform stage
	terraformOptions := test_structure.LoadTerraformOptions(t, workingDir)

	// Load the keypair we saved before we ran terraform
	keyPair := test_structure.LoadEc2KeyPair(t, workingDir)

	// Run `terraform output` to get the value of an output variable
	instanceIPAddress := terraform.Output(t, terraformOptions, "openvpn_host_public_ip")

	// Host details used to connect to the host
	host := ssh.Host{
		Hostname:    instanceIPAddress,
		SshUserName: osToSSHUserName(t, osName),
		SshKeyPair:  keyPair.KeyPair,
	}

	// It can take a minute or so for the Instance to boot up, so retry a few times
	maxRetries := 30

	waitUntilSSHAvailable(t, host, maxRetries, 5*time.Second)
	waitUntilOpenVpnInitComplete(t, host, maxRetries, 30*time.Second)

	logger.Log(t, "SetupSuite Complete, Running Tests")

	t.Run("openvpn tests", func(t *testing.T) {
		t.Run("running testOpenVpnIsRunning", wrapTestCase(testOpenVpnIsRunning, host))
		t.Run("running testOpenVpnAdminProcessRequestsIsRunning", wrapTestCase(testOpenVpnAdminProcessRequestsIsRunning, host))
		t.Run("running testOpenVpnAdminProcessRevokesIsRunning", wrapTestCase(testOpenVpnAdminProcessRevokesIsRunning, host))
		t.Run("running testCrlExpirationDateUpdated", wrapTestCase(testCrlExpirationDateUpdated, host))
		t.Run("running testCronJobExists", wrapTestCase(testCronJobExists, host))
	})
}

func testOpenVpnIsRunning(t *testing.T, host ssh.Host) {
	commandToTest := "sudo ps -ef|grep openvpn"
	output := ssh.CheckSshCommand(t, host, commandToTest)

	// It will be convenient to see the full command output directly in logs. This will show only when there's a test failure.
	logger.Logf(t, "Result of running \"%s\"\n", commandToTest)
	logger.Log(t, output)

	assert.Contains(t, output, "/usr/sbin/openvpn")
}

func testOpenVpnAdminProcessRequestsIsRunning(t *testing.T, host ssh.Host) {
	commandToTest := "sudo ps -ef|grep openvpn"
	output := ssh.CheckSshCommand(t, host, commandToTest)

	// It will be convenient to see the full command output directly in logs. This will show only when there's a test failure.
	logger.Logf(t, "Result of running \"%s\"\n", commandToTest)
	logger.Log(t, output)

	assert.Contains(t, output, "/usr/local/bin/openvpn-admin process-requests")
}

func testOpenVpnAdminProcessRevokesIsRunning(t *testing.T, host ssh.Host) {
	commandToTest := "sudo ps -ef|grep openvpn"
	var err error
	output := ssh.CheckSshCommand(t, host, commandToTest)
	if err != nil {
		t.Fatalf("Failed to SSH to AMI Builder at %s and execute command :%s\n", host.Hostname, err.Error())
	}

	// It will be convenient to see the full command output directly in logs. This will show only when there's a test failure.
	logger.Logf(t, "Result of running \"%s\"\n", commandToTest)
	logger.Log(t, output)

	assert.Contains(t, output, "/usr/local/bin/openvpn-admin process-revokes")
}

func testCronJobExists(t *testing.T, host ssh.Host) {
	commandToTest := "sudo cat /etc/cron.hourly/backup-openvpn-pki"
	output := ssh.CheckSshCommand(t, host, commandToTest)

	// It will be convenient to see the full command output directly in logs. This will show only when there's a test failure.
	logger.Logf(t, "Result of running \"%s\"\n", commandToTest)
	logger.Log(t, output)

	assert.Contains(t, output, "backup-openvpn-pki")
}

func testCrlExpirationDateUpdated(t *testing.T, host ssh.Host) {
	commandToTest := "sudo cat /etc/openvpn-ca/openssl-1.0.0.cnf"
	output := ssh.CheckSshCommand(t, host, commandToTest)

	// It will be convenient to see the full command output directly in logs. This will show only when there's a test failure.
	logger.Logf(t, "Result of running \"%s\"\n", commandToTest)
	logger.Log(t, output)

	assert.Contains(t, output, "default_crl_days= 3650")
}

func wrapTestCase(testCase func(t *testing.T, host ssh.Host), host ssh.Host) func(t *testing.T) {
	return func(t *testing.T) {
		testCase(t, host)
	}
}

func waitUntilSSHAvailable(t *testing.T, host ssh.Host, maxRetries int, timeBetweenRetries time.Duration) {

	retry.DoWithRetry(
		t,
		fmt.Sprintf("SSH to public host %s", host.Hostname),
		maxRetries,
		timeBetweenRetries,
		func() (string, error) {
			return "", ssh.CheckSshConnectionE(t, host)
		},
	)

}

func waitUntilOpenVpnInitComplete(t *testing.T, host ssh.Host, maxRetries int, timeBetweenRetries time.Duration) {
	retry.DoWithRetry(
		t,
		fmt.Sprintf("Waiting for OpenVPN initialization to complete"),
		maxRetries,
		timeBetweenRetries,
		func() (string, error) {
			return "", initComplete(t, host)
		},
	)
}

func initComplete(t *testing.T, host ssh.Host) error {
	command := "sudo ls /etc/openvpn/openvpn-init-complete"
	output, err := ssh.CheckSshCommandE(t, host, command)

	if strings.Contains(output, "such file or directory") {
		return fmt.Errorf("OpenVPN initialization not yet complete")
	}

	if err != nil {
		return err
	}

	return nil
}

func osToSSHUserName(t *testing.T, osName string) string {
	if strings.Contains(osName, "ubuntu") {
		return "ubuntu"
	}
	t.Fatalf("Unknown osName - can't map the os (%s) to it's default user", osName)
	return ""
}

func osToLogPathSpec(t *testing.T, osName string) map[string][]string {
	if strings.Contains(osName, "ubuntu") {
		return map[string][]string{
			"/var/log": []string{"cloud-init.log", "cloud-init-output.log", "syslog"},
		}
	}
	t.Fatalf("Unknown osName - can't map the os (%s) to it's default user", osName)
	return nil
}
