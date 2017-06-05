package test

import (
	"log"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pquerna/otp/totp"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sts"
)

// Create a new AWS session using the system environment to authenticate to AWS. For info on how to configure the system
// environment, see https://docs.aws.amazon.com/sdk-for-go/v1/developerguide/configuring-sdk.html.
func createAwsSession(t *testing.T, awsRegion string) *session.Session {
	awsConfig := defaults.Get().Config.WithRegion(awsRegion)

	_, err := awsConfig.Credentials.Get()
	if err != nil {
		t.Fatalf("Error finding AWS credentials (did you set the AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables?): %s", err)
	}

	return session.New( awsConfig )
}

// Create a new AWS session using explicit credentials. This is useful if you want to create an IAM User dynamically and
// create an AWS session authenticated as the new IAM User.
func createAwsSessionWithCreds(t *testing.T, awsRegion string, accessKeyId string, secretAccessKey string) *session.Session {
	creds := createAwsCredentials(accessKeyId, secretAccessKey)
	awsConfig := defaults.Get().Config.WithRegion(awsRegion).WithCredentials(creds)
	return session.New( awsConfig )
}

func createAwsSessionWithCredsAndMfa(t *testing.T, awsRegion string, stsClient *sts.STS, iamClient *iam.IAM, mfaDevice *iam.VirtualMFADevice) *session.Session {
	// Get a one-time password
	tokenCode := getTimeBasedOneTimePassword(t, mfaDevice)

	// Now get temp credentials from STS
	output, err := stsClient.GetSessionToken(&sts.GetSessionTokenInput{
		SerialNumber: mfaDevice.SerialNumber,
		TokenCode: aws.String(tokenCode),
	})
	if err != nil {
		t.Fatalf("Error getting session token: %s\n", err)
	}

	accessKeyId := *output.Credentials.AccessKeyId
	secretAccessKey := *output.Credentials.SecretAccessKey
	sessionToken := *output.Credentials.SessionToken

	// Now authenticate a session with MFA
	creds := createAwsCredentialsWithSessionToken(accessKeyId, secretAccessKey, sessionToken)
	awsConfig := defaults.Get().Config.WithRegion(awsRegion).WithCredentials(creds)
	return session.New( awsConfig )
}

// Create an AWS configuration with specific AWS credentials.
func createAwsCredentials(accessKeyId string, secretAccessKey string) *credentials.Credentials {
	creds := credentials.Value{ AccessKeyID: accessKeyId, SecretAccessKey: secretAccessKey }
	return credentials.NewStaticCredentialsFromCreds(creds)
}

// Create an AWS configuration with temporary AWS credentials by including a session token (used for authenticating with MFA)
func createAwsCredentialsWithSessionToken(accessKeyId, secretAccessKey, sessionToken string) *credentials.Credentials {
	creds := credentials.Value{
		AccessKeyID: accessKeyId,
		SecretAccessKey: secretAccessKey,
		SessionToken: sessionToken,
	}
	return credentials.NewStaticCredentialsFromCreds(creds)
}

func createMfaDevice(t *testing.T, logger *log.Logger, iamClient *iam.IAM, deviceName string) *iam.VirtualMFADevice {
	output, err := iamClient.CreateVirtualMFADevice(&iam.CreateVirtualMFADeviceInput{
		VirtualMFADeviceName: aws.String(deviceName),
	})
	if err != nil {
		t.Fatalf("Error creating Virtual MFA Device: %s\n", err)
	}

	mfaDevice := output.VirtualMFADevice

	enableMfaDevice(t, logger, iamClient, mfaDevice)

	return mfaDevice
}

// Enable a newly created MFA Device (by supplying the first two one-time passwords) so that it can be used for future
// logins by the given IAM User
func enableMfaDevice(t *testing.T, logger *log.Logger, iamClient *iam.IAM, mfaDevice *iam.VirtualMFADevice) {
	iamUserName := getUserName(t, iamClient)

	authCode1 := getTimeBasedOneTimePassword(t, mfaDevice)

	logger.Println("Waiting 30 seconds for a new MFA Token to be generated...")
	time.Sleep(30 * time.Second)

	authCode2 := getTimeBasedOneTimePassword(t, mfaDevice)

	_, err := iamClient.EnableMFADevice(&iam.EnableMFADeviceInput{
		AuthenticationCode1: aws.String(authCode1),
		AuthenticationCode2: aws.String(authCode2),
		SerialNumber: mfaDevice.SerialNumber,
		UserName: aws.String(iamUserName),
	})
	if err != nil {
		t.Fatalf("Failed to enable MFA device: %s\n", err)
	}

	sleepWithMessage(logger, 10 * time.Second, "Waiting for MFA Device enablement to propagate.")
}

// Get the user name of the given IAM Client session
func getUserName(t *testing.T, iamClient *iam.IAM) string {
	output, err := iamClient.GetUser(&iam.GetUserInput{})
	if err != nil {
		t.Fatalf("Error getting IAM User Name; %s\n", err)
	}

	return *output.User.UserName
}

// Get a One-Time Password from the given mfaDevice. Per the RFC 6238 standard, this value will be different every 30 seconds.
func getTimeBasedOneTimePassword(t *testing.T, mfaDevice *iam.VirtualMFADevice) string {
	base32StringSeed := string(mfaDevice.Base32StringSeed)

	otp, err := totp.GenerateCode(base32StringSeed, time.Now())
	if err != nil {
		t.Fatalf("Failed to generate a time-based one-time password: %s\n", err)
	}

	return otp
}

// Get the AWS Account ID of the currently authenticated session.
func getAwsAccountId(t *testing.T, iamClient *iam.IAM) string {
	// By leaving the "Username" property of GetUserInput blank, we will get information on the IAM User making the request.
	output, err := iamClient.GetUser(&iam.GetUserInput{})
	if err != nil {
		t.Fatalf("Error getting information on current IAM User: %s", err)
	}

	awsAccountId := extractAwsAccountIdFromIamUserArn(*output.User.Arn)
	return awsAccountId
}

func extractAwsAccountIdFromIamUserArn(iamUserArn string) string {
	arnSlice := strings.Split(iamUserArn, ":")
	return arnSlice[4]
}

func readPasswordPolicyMinPasswordLength(t *testing.T, iamClient *iam.IAM) int {
	output, err := iamClient.GetAccountPasswordPolicy(&iam.GetAccountPasswordPolicyInput{})
	if err != nil {
		t.Fatalf("Failed to get IAM User Password Policy: %v", err)
	}

	return int(*output.PasswordPolicy.MinimumPasswordLength)
}

// Create a client the SDK can use to perform operations on the EC2 service.
func createEc2Client(session *session.Session) *ec2.EC2 {
	return ec2.New(session)
}

// Create a client the SDK can use to perform operations on the IAM service.
func createIamClient(session *session.Session) *iam.IAM {
	return iam.New(session)
}

// Create a client the SDK can use to perform operations on the KMS service.
func createKmsClient(session *session.Session) *kms.KMS {
	return kms.New(session)
}

// Create a client the SDK can use to perform operations on the S3 service.
func createS3Client(session *session.Session) *s3.S3 {
	return s3.New(session)
}

// Create a client the SDK can use to perform operations on the STs service.
func createStsClient(session *session.Session) *sts.STS {
	return sts.New(session)
}
