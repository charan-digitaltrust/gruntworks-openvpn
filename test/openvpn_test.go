package test

import (
	"testing"
	"github.com/gruntwork-io/terratest/packer"
	"log"
	"github.com/stretchr/testify/assert"
	"github.com/gruntwork-io/terratest"
	terralog "github.com/gruntwork-io/terratest/log"
	"github.com/stretchr/testify/suite"
	"fmt"
	"time"
	"github.com/gruntwork-io/terratest/ssh"
	"github.com/gruntwork-io/terratest/util"
	"strings"
	"strconv"
)

const TEST_NAME = "TestOpenVPN"

type openVpnTestSuite struct {
	suite.Suite
	logger             *log.Logger
	amiId              string
	resourceCollection *terratest.RandomResourceCollection
	terratestOptions   *terratest.TerratestOptions
	output             string
	ipAddress          string
	host               ssh.Host
}

func getRandomizedString(prefix string) (string) {
	return fmt.Sprintf("%s-%s", prefix, strconv.FormatInt(time.Now().Unix(), 10))
}

func (suite *openVpnTestSuite) SetupSuite() {
	suite.logger = terralog.NewLogger(TEST_NAME)
	suite.resourceCollection = createBaseRandomResourceCollection(suite.T())

	//Build the AMI with packer
	packerOptions := packer.PackerOptions{
		Template: "../examples/packer/openvpn-server-ubuntu1604.json",
		Vars: map[string]string{
			"aws_region": suite.resourceCollection.AwsRegion,
			"package_openvpn_branch": getCurrentBranchName(suite.T()),
			"active_git_branch": getCurrentBranchName(suite.T()),
		},
	}

	var err error
	suite.amiId, err = packer.BuildAmi(packerOptions, suite.logger)
	if err != nil {
		suite.Fail("Failed to build AMI due to error: %s", err.Error())
	}

	if suite.amiId == "" {
		suite.Fail("Got a blank AMI Id for template %s", packerOptions.Template)
	}

	vpc, err := suite.resourceCollection.GetDefaultVpc()
	if err != nil {
		suite.T().Fatalf("Failed to get default VPC: %s\n", err.Error())
	}
	if len(vpc.Subnets) == 0 {
		suite.T().Fatalf("Default vpc %s contained no subnets", vpc.Id)
	}

	//Setup terratest
	suite.terratestOptions = createBaseTerratestOptions(suite.T(), TEST_NAME, "../examples/openvpn-host", suite.resourceCollection)

	suite.terratestOptions.Vars["name"] = getRandomizedString("tst-openvpn-host")
	suite.terratestOptions.Vars["backup_bucket_name"] = getRandomizedString("tst-openvpn")
	suite.terratestOptions.Vars["request_queue_name"] = getRandomizedString("tst-openvpn-requests")
	suite.terratestOptions.Vars["revocation_queue_name"] = getRandomizedString("tst-openvpn-revocations")
	suite.terratestOptions.Vars["ami"] = suite.amiId
	suite.terratestOptions.Vars["keypair_name"] = suite.resourceCollection.KeyPair.Name

	suite.output, err = terratest.Apply(suite.terratestOptions)
	assert.Nil(suite.T(), err, "Unexpected error when applying terraform templates: %v", err)

	suite.ipAddress, err = terratest.Output(suite.terratestOptions, "openvpn_host_public_ip")
	if err != nil {
		suite.logger.Fatal(fmt.Sprintf("An error occurred retreiving terraform output - %s", err.Error()))
	}

	// SSH into EC2 Instance
	suite.host = ssh.Host{
		Hostname: suite.ipAddress,
		SshUserName: "ubuntu",
		SshKeyPair: suite.resourceCollection.KeyPair,
	}

	_, err = util.DoWithRetry(
		fmt.Sprintf("SSH to public host %s", suite.ipAddress),
		10,
		30 * time.Second,
		suite.logger,
		func() (string, error) {
			return "", ssh.CheckSshConnection(suite.host, suite.logger)
		},
	)

	if err != nil {
		suite.T().Fatalf("Failed to SSH to host at %s and execute command :%s\n", suite.ipAddress, err.Error())
	}

	_, err = util.DoWithRetry(
		fmt.Sprintf("Waiting for OpenVPN initialization to complete"),
		60,
		30 * time.Second,
		suite.logger,
		func() (string, error) {
			return "", initComplete(suite)
		},
	)

	if err != nil {
		suite.T().Fatalf("OpenVPN initilization failed to complete")
	}

	suite.logger.Println("SetupSuite Complete")
}

func (suite *openVpnTestSuite) TearDownSuite() {
	terratest.Destroy(suite.terratestOptions, suite.resourceCollection)

	session := createAwsSession(suite.T(), suite.resourceCollection.AwsRegion)
	ec2Client := createEc2Client(session)

	removeAmi(suite.T(), ec2Client, suite.amiId, suite.logger)
	suite.logger.Println("TearDownSuite Complete")
}

func initComplete(suite *openVpnTestSuite) (error) {
	command := "sudo ls /etc/openvpn/openvpn-init-complete"
	output, err := ssh.CheckSshCommand(suite.host, command, suite.logger)
	if err != nil {
		return err
	}

	if strings.Contains(output, "no such file or directory") {
		return fmt.Errorf("initialization not yet complete")
	}

	return nil
}

func (suite *openVpnTestSuite) TestOpenVpnIsRunning() {
	commandToTest := "sudo ps -ef|grep openvpn"
	output, err := ssh.CheckSshCommand(suite.host, commandToTest, suite.logger)
	if err != nil {
		suite.T().Fatalf("Failed to SSH to AMI Builder at %s and execute command :%s\n", suite.ipAddress, err.Error())
	}

	// It will be convenient to see the full command output directly in logs. This will show only when there's a test failure.
	suite.T().Logf("Result of running \"%s\"\n", commandToTest)
	suite.T().Log(output)

	assert.Contains(suite.T(), output, "/usr/sbin/openvpn")
}

func (suite *openVpnTestSuite) TestOpenVpnAdminProcessCertsIsRunning() {
	commandToTest := "sudo ps -ef|grep openvpn"
	output, err := ssh.CheckSshCommand(suite.host, commandToTest, suite.logger)
	if err != nil {
		suite.T().Fatalf("Failed to SSH to AMI Builder at %s and execute command :%s\n", suite.ipAddress, err.Error())
	}

	// It will be convenient to see the full command output directly in logs. This will show only when there's a test failure.
	suite.T().Logf("Result of running \"%s\"\n", commandToTest)
	suite.T().Log(output)

	assert.Contains(suite.T(), output, "openvpn-admin process-requests")
}

func TestOpenVpnSuite(t *testing.T) {
	tests := new(openVpnTestSuite)
	suite.Run(t, tests)
}