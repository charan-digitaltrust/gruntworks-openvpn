package openvpn

import (
	"github.com/gruntwork-io/package-openvpn/modules/openvpn-admin/src/aws_helpers"
)

var s3BackupBucket string

func getBackupBucketName(awsRegion string) (string, error) {
	if s3BackupBucket == "" {
		bucketName, err := aws_helpers.FindS3BucketWithTag(awsRegion, "OpenVPNRole", "BackupBucket")
		if err != nil {
			return "", err
		}
		s3BackupBucket = bucketName
	}
	return s3BackupBucket, nil
}

func GetRequestQueueUrl(awsRegion string) (string, error) {
	bucket, err := getBackupBucketName(awsRegion)
	if err != nil {
		return "", err
	}

	return aws_helpers.GetS3ObjectContents(awsRegion, bucket, "client/request-queue-url")
}

func GetRevokeQueueUrl(awsRegion string) (string, error) {
	bucket, err := getBackupBucketName(awsRegion)
	if err != nil {
		return "", err
	}

	return aws_helpers.GetS3ObjectContents(awsRegion, bucket, "client/revoke-queue-url")
}