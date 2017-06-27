package app

import (
	"github.com/urfave/cli"
	"fmt"
	"github.com/gruntwork-io/gruntwork-cli/logging"
	"encoding/json"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/gruntwork-io/package-openvpn/modules/openvpn-admin/src/aws_helpers"
)

type CertificateRevokeRequest struct {
	Username      string
	ResponseQueue string
}

type CertificateRevokeResponse struct {
	Success      bool
	ErrorMessage string
}

func requestCertificateRevocation(cliContext *cli.Context) error {
	setLoggerLevel(cliContext)
	logger := logging.GetLogger(LOGGER_NAME)

	awsRegion, err := getAwsRegion(cliContext)
	if err != nil {
		return err
	}
	logger.Debugf("Using AWS Region: %s", awsRegion)

	username, err := getUsername(cliContext, false)
	if err != nil {
		return err
	}
	logger.Debugf("Using Username: %s", username)

	logger.Info("Looking up SQS queue")
	revokeUrl, err := getRevokeUrl(cliContext)
	if err != nil {
		return err
	}
	logger.Debugf("Using Revoke URL: %s", revokeUrl)

	timeout, err := getTimeout(cliContext)
	if err != nil {
		return err
	}

	//Create a new response queue
	logger.Info("Creating temporary SQS response queue")
	responseQueue, err := createResponseQueue(awsRegion)
	if err != nil {
		return err
	}
	defer deleteResponseQueue(awsRegion, responseQueue)

	//Put a request for a new certificate revocation on the revokeQueue
	logger.Infof("Requesting certificate revocation for %s on %s", username, revokeUrl)
	err = sendRevoke(awsRegion, revokeUrl, username, responseQueue)
	if err != nil {
		return err
	}

	// Wait for a reply from OpenVPN server on the responseQueue
	logger.Info("Waiting for response from OpenVPN server")
	receipt, response, err := waitForMessage(awsRegion, responseQueue, timeout)
	if err != nil {
		return err
	}

	// Process the response
	logger.Info("Response received from OpenVPN server")
	err = processRevokeResponse(awsRegion, responseQueue, receipt, response, username)
	if err != nil {
		return err
	}

	logger.Info("DONE")
	return nil
}

func sendRevoke(awsRegion string, revokeQueue string, username string, responseQueue string) error {
	req := &CertificateRevokeRequest{
		Username: username,
		ResponseQueue:responseQueue,
	}
	requestJson, _ := json.Marshal(req)

	err := aws_helpers.SendMessageToQueue(awsRegion, revokeQueue, string(requestJson))
	if err != nil {
		return err
	}
	return nil
}

func processRevokeResponse(awsRegion string, responseQueue string, receipt string, message string, username string) error {
	response := CertificateRevokeResponse{}
	json.Unmarshal([]byte(message), &response)

	if (!response.Success) {
		aws_helpers.DeleteMessageFromQueue(awsRegion, responseQueue, receipt)
		return errors.WithStackTrace(fmt.Errorf(response.ErrorMessage))
	}

	aws_helpers.DeleteMessageFromQueue(awsRegion, responseQueue, receipt)
	return nil
}

