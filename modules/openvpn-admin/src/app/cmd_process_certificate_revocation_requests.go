package app

import (
	"github.com/urfave/cli"
	"encoding/json"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/gruntwork-io/gruntwork-cli/logging"
	"fmt"
	"github.com/gruntwork-io/package-openvpn/modules/openvpn-admin/src/aws_helpers"
)

// NOTE: This method runs in an infinite loop
func processCertificateRevocationRequests(cliContext *cli.Context) error {
	setLoggerLevel(cliContext)
	logger := logging.GetLogger(LOGGER_NAME)

	awsRegion, err := getAwsRegion(cliContext)
	if err != nil {
		return err
	}
	logger.Debugf("Using AWS Region: %s", awsRegion)

	revokeUrl, err := getRevokeUrl(cliContext)
	if err != nil {
		return err
	}
	logger.Debugf("Using Revoke URL: %s", revokeUrl)

	timeout, err := getTimeout(cliContext)
	if err != nil {
		return err
	}

	for {
		// Wait for a request to come in from a client on the revokeQueue
		receipt, revokeRequest, err := waitForMessage(awsRegion, revokeUrl, timeout)
		if err != nil {
			if sleepOnFailedToReceiveMessages(err) {
				continue
			}
			return err
		}

		//Here if we encounter an error, we don't want to stop processing, we want to return the error to the caller
		//via the SQS queue
		responseQueue, err := processRevokeRequest(awsRegion, receipt, revokeRequest)
		if err != nil {
			logger.WithError(err)
		}

		err = sendRevokeReply(awsRegion, responseQueue, err)
		if err != nil {
			return err
		}

		err = aws_helpers.DeleteMessageFromQueue(awsRegion, revokeUrl, receipt)
		if err != nil {
			return err
		}

		logger.Info("DONE")
	}
	return nil
}

func processRevokeRequest(awsRegion string, receipt string, message string) (string, error) {

	revokeRequest := CertificateRevokeRequest{}
	json.Unmarshal([]byte(message), &revokeRequest)

	certificateAlreadyExists, err := indexContainsValidCertificate(revokeRequest.Username)
	if err != nil {
		return revokeRequest.ResponseQueue, err
	}

	if (certificateAlreadyExists) {
		err := revokeCertificate(revokeRequest.Username)
		if err != nil {
			return revokeRequest.ResponseQueue, err
		}

		return revokeRequest.ResponseQueue, nil
	} else {
		var doesNotExistError = fmt.Errorf("a valid certificate for %s does not exist", revokeRequest.Username)
		return revokeRequest.ResponseQueue, errors.WithStackTrace(doesNotExistError)
	}

}

func sendRevokeReply(awsRegion string, responseQueue string, error error) error {
	logger := logging.GetLogger(LOGGER_NAME)

	responseMessage := &CertificateRevokeResponse{}
	responseMessage.Success = (error == nil)

	if !responseMessage.Success {
		responseMessage.ErrorMessage = error.Error()
	}

	responseJson, err := json.Marshal(responseMessage)
	if err != nil {
		return err
	}

	logger.Debugf("Sending revocation reply %s on %s", string(responseJson), responseQueue)

	err = aws_helpers.SendMessageToQueue(awsRegion, responseQueue, string(responseJson))
	if err != nil {
		return err
	}
	return nil
}

