package aws_helpers

import (
	"github.com/aws/aws-sdk-go/service/s3"
	"strings"
	"github.com/gruntwork-io/gruntwork-cli/logging"
	"bytes"
)

func CreateS3Client(awsRegion string) (*s3.S3, error) {
	sess, err := CreateAwsSession(awsRegion, NO_IAM_ROLE)
	if err != nil {
		return nil, err
	}

	return s3.New(sess), nil
}

func FindS3BucketWithTag(awsRegion string, key string, value string) (string, error) {
	logger := logging.GetLogger(LOGGER_NAME)

	s3Client, err := CreateS3Client(awsRegion)
	if err != nil {
		return "", err
	}

	resp, err := s3Client.ListBuckets(&s3.ListBucketsInput{})
	if err != nil {
		return "", err
	}

	for _, bucket := range resp.Buckets {
		tagResponse, err := s3Client.GetBucketTagging(&s3.GetBucketTaggingInput{Bucket: bucket.Name})
		if err != nil {

			if !strings.Contains(err.Error(), "AuthorizationHeaderMalformed") &&
				!strings.Contains(err.Error(), "BucketRegionError") &&
				!strings.Contains(err.Error(), "NoSuchTagSet") {
				return "", err
			}

		}

		for _, tag := range tagResponse.TagSet {
			if *tag.Key == key && *tag.Value == value {
				logger.Debugf("Found S3 bucket %s with %s=%s", *bucket.Name, key, value)
				return *bucket.Name, nil
			}
		}
	}

	return "", nil
}

func GetS3ObjectContents(awsRegion string, bucket string, key string) (string, error) {
	logger := logging.GetLogger(LOGGER_NAME)

	s3Client, err := CreateS3Client(awsRegion)
	if err != nil {
		return "", err
	}

	res, err := s3Client.GetObject(&s3.GetObjectInput{
		Bucket: &bucket,
		Key: &key,
	})

	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(res.Body)
	if err != nil {
		return "", err
	}

	contents := buf.String()
	logger.Debugf("Read contents from s3://%s/%s", bucket, key)

	return contents, nil
}
