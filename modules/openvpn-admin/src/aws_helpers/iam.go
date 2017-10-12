package aws_helpers

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
)

type PolicyDocument struct {
	Version   string
	Statement []StatementEntry
}

type StatementEntry struct {
	Effect   string
	Action   []string
	Resource string
}

// A convenience variable that gives you a readable way to specify you don't need an IAM role for the current operation.
const NO_IAM_ROLE = ""

// Create an AWS Session object in the given region and check that credentials are present. If roleArn is not empty,
// assume the specified IAM role.
func CreateAwsSession(awsRegion string, roleArn string) (*session.Session, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	sess.Config.Region = aws.String(awsRegion)

	if roleArn != "" {
		sess.Config.Credentials = stscreds.NewCredentials(sess, roleArn)
	}

	if _, err := sess.Config.Credentials.Get(); err != nil {
		return nil, errors.WithStackTraceAndPrefix(err, "Error finding AWS credentials (did you set the AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables?)")
	}

	return sess, nil
}

func GetIamUserName(awsRegion string) (string, error) {

	iamClient, err := createIamClient(awsRegion)
	if err != nil {
		return "", err
	}

	resp, err := iamClient.GetUser(&iam.GetUserInput{})
	if err != nil {
		return "", err
	}

	return *resp.User.UserName, nil
}

func createIamClient(awsRegion string) (*iam.IAM, error) {
	sess, err := CreateAwsSession(awsRegion, NO_IAM_ROLE)
	if err != nil {
		return nil, err
	}

	return iam.New(sess), nil
}