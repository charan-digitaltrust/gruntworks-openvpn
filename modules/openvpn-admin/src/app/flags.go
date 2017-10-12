package app

import (
	"github.com/urfave/cli"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	valid "github.com/asaskevich/govalidator"
	"github.com/gruntwork-io/gruntwork-cli/logging"
	"github.com/sirupsen/logrus"
	"github.com/gruntwork-io/package-openvpn/modules/openvpn-admin/src/aws_helpers"
	"fmt"
)

const REQUEST_QUEUE_NAME_PREFIX = "openvpn-requests-"
const REVOCATION_QUEUE_NAME_PREFIX = "openvpn-revocations-"

func setLoggerLevel(cliContext *cli.Context) () {
	debug := cliContext.Bool(OPTION_DEBUG)
	if (debug) {
		logging.SetGlobalLogLevel(logrus.DebugLevel)
	}
}

func getAwsRegion(cliContext *cli.Context) (string, error) {
	awsRegion := cliContext.String(OPTION_AWS_REGION)
	if awsRegion == "" {
		return "", errors.WithStackTrace(MissingAwsRegion)
	}
	return awsRegion, nil
}

func getUsername(cliContext *cli.Context, allowSearch bool) (string, error) {
	var userName string
	var err error

	awsRegion, err := getAwsRegion(cliContext)
	if err != nil {
		return "", errors.WithStackTrace(err)
	}

	userName = cliContext.String(OPTION_USERNAME)
	if userName == "" && allowSearch {
		// if userName flag is empty, try to get from IAM user
		userName, err = aws_helpers.GetIamUserName(awsRegion)
		if err != nil {
			return "", errors.WithStackTrace(err)
		}

		if userName == "" {
			return "", errors.WithStackTrace(MissingUsername)
		}
	}

	if userName == "" {
		return "", errors.WithStackTrace(MissingUsername)
	}

	return userName, nil
}

func getTimeout(cliContext *cli.Context) (int, error) {
	timeout := cliContext.Int(OPTION_TIMEOUT)
	return timeout, nil
}

func getRequestUrl(cliContext *cli.Context) (string, error) {
	var url string
	var err error

	logger := logging.GetLogger(LOGGER_NAME)

	awsRegion, err := getAwsRegion(cliContext)
	if err != nil {
		return "", errors.WithStackTrace(err)
	}

	url = cliContext.String(OPTION_REQUEST_URL)

	if url == "" {
		logger.Debug("Locating Request URL in " + awsRegion)
		// if url flag is empty, try to get it automatically based on naming conventions
		url, err = getQueueUrl(awsRegion, REQUEST_QUEUE_NAME_PREFIX, OPTION_REQUEST_URL)
		if err != nil {
			return "", errors.WithStackTrace(err)
		}

		if url == "" {
			return "", errors.WithStackTrace(MissingRequestUrl)
		}
	} else {
		logger.Debugf("Using Request URL from flags %s ", url)
	}

	if !valid.IsURL(url) {
		return "", errors.WithStackTrace(MissingRequestUrl)
	}

	return url, nil
}

func getRevokeUrl(cliContext *cli.Context) (string, error) {
	var url string
	var err error

	logger := logging.GetLogger(LOGGER_NAME)

	awsRegion, err := getAwsRegion(cliContext)
	if err != nil {
		return "", errors.WithStackTrace(err)
	}

	url = cliContext.String(OPTION_REVOKE_URL)
	if url == "" {
		logger.Debugf("Locating Revoke URL in %s", awsRegion)

		// if url flag is empty, try to get it automatically based on naming conventions
		url, err = getQueueUrl(awsRegion, REVOCATION_QUEUE_NAME_PREFIX, OPTION_REVOKE_URL)
		if err != nil {
			return "", errors.WithStackTrace(err)
		}

		if url == "" {
			return "", errors.WithStackTrace(MissingRevokeUrl)
		}
	} else {
		logger.Debugf("Using Revoke URL from flags %s", url)
	}

	if !valid.IsURL(url) {
		return "", errors.WithStackTrace(MissingRevokeUrl)
	}

	return url, nil
}

func getQueueUrl(awsRegion string, queueNamePrefix string, argName string) (string, error) {
	queueUrls, err := aws_helpers.FindQueuesWithNamePrefix(awsRegion, queueNamePrefix)
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