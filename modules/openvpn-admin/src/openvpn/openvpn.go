package openvpn

import (
	"github.com/gruntwork-io/package-openvpn/modules/openvpn-admin/src/aws_helpers"
	"fmt"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/gruntwork-io/package-openvpn/modules/openvpn-admin/src/app"
)

const REQUEST_QUEUE_NAME_PREFIX = "openvpn-requests-"
const REVOCATION_QUEUE_NAME_PREFIX = "openvpn-revocations-"

func getQueueUrl(awsRegion string, roleArn string, queueNamePrefix string, argName string) (string, error) {
	queueUrls, err := aws_helpers.FindQueuesWithNamePrefix(awsRegion, roleArn, queueNamePrefix)
	if err != nil {
		return "", err
	}
	if len(queueUrls) == 0 {
		return "", errors.WithStackTrace(NoQueuesFoundWithPrefix(queueNamePrefix))
	}
	if len(queueUrls) > 1 {
		return "", errors.WithStackTrace(MultipleQueuesFoundWithPrefix{Prefix: queueNamePrefix, QueueUrls: queueUrls, ArgName: argName})
	}
	return queueUrls[0], nil
}

func GetRequestQueueUrl(awsRegion string, roleArn string) (string, error) {
	return getQueueUrl(awsRegion, roleArn, REQUEST_QUEUE_NAME_PREFIX, app.OPTION_REQUEST_URL)
}

func GetRevokeQueueUrl(awsRegion string, roleArn string) (string, error) {
	return getQueueUrl(awsRegion, roleArn, REVOCATION_QUEUE_NAME_PREFIX, app.OPTION_REVOKE_URL)
}

// Custom errors

type NoQueuesFoundWithPrefix string
func (err NoQueuesFoundWithPrefix) Error() string {
	return fmt.Sprintf("Could not find any SQS queues with the name prefix '%s'.", string(err))
}

type MultipleQueuesFoundWithPrefix struct {
	Prefix    string
	QueueUrls []string
	ArgName   string
}
func (err MultipleQueuesFoundWithPrefix) Error() string {
	return fmt.Sprintf("Expected to find exactly one queue with prefix '%s' but found %d: %v. Please specify which queue URL to use using the %s argument.", err.Prefix, len(err.QueueUrls), err.QueueUrls, err.ArgName)
}