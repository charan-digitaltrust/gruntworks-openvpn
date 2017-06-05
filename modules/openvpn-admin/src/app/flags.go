package app

import (
	"github.com/urfave/cli"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	valid "github.com/asaskevich/govalidator"
	"github.com/gruntwork-io/gruntwork-cli/logging"
	"github.com/Sirupsen/logrus"
	"github.com/gruntwork-io/package-openvpn/modules/openvpn-admin/src/aws_helpers"
	"github.com/gruntwork-io/package-openvpn/modules/openvpn-admin/src/openvpn"
)

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
		// if url flag is empty, try to get from S3 bucket
		url, err = openvpn.GetRequestQueueUrl(awsRegion)
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

		// if url flag is empty, try to get from S3 bucket
		url, err = openvpn.GetRevokeQueueUrl(awsRegion)
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
