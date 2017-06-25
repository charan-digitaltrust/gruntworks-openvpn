package test

import (
	"log"
	"os/exec"
	"strings"
	"testing"
	"time"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/terratest"
)

func createBaseTerratestOptions(t *testing.T, testName string, templatePath string, resourceCollection *terratest.RandomResourceCollection) *terratest.TerratestOptions {
	terratestOptions := terratest.NewTerratestOptions()

	terratestOptions.UniqueId = resourceCollection.UniqueId
	terratestOptions.TemplatePath = templatePath
	terratestOptions.TestName = testName

	terratestOptions.Vars = map[string]interface{}{
		"aws_region":     resourceCollection.AwsRegion,
	}

	return terratestOptions
}

func removeAmi(t *testing.T, ec2Client *ec2.EC2, imageId string, logger *log.Logger) {
	t.Logf("Deregistering AMI %s\n", imageId)

	_, err := ec2Client.DeregisterImage(&ec2.DeregisterImageInput{ImageId: aws.String(imageId) })
	if err != nil {
		t.Fatalf("Got unexpected error deregistering AMI %s: %v\n", imageId, err)
	}
}

func getCurrentBranchName(t *testing.T) string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	bytes, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to determine current branch name due to error: %s\n", err)
	}
	return strings.TrimSpace(string(bytes))
}

func sleepWithMessage(logger *log.Logger, duration time.Duration, whySleepMessage string) {
	logger.Printf("Sleeping %v: %s\n", duration, whySleepMessage)
	time.Sleep(duration)
}