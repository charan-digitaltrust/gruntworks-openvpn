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
func processNewCertificateRequests(cliContext *cli.Context) error {
	setLoggerLevel(cliContext)
	logger := logging.GetLogger(LOGGER_NAME)

	awsRegion, err := getAwsRegion(cliContext)
	if err != nil {
		return err
	}
	logger.Debugf("Using AWS Region: %s", awsRegion)

	roleArn, err := getRoleArn(cliContext)
	if err != nil {
		return err
	}

	requestUrl, err := getRequestUrl(cliContext, roleArn)
	if err != nil {
		return err
	}
	logger.Debugf("Using Request URL: %s", requestUrl)

	timeout, err := getTimeout(cliContext)
	if err != nil {
		return err
	}

	for {
		// Wait for a request to come in from a client on the requestQueue
		receipt, request, err := waitForRequestMessage(awsRegion, roleArn, requestUrl, timeout)
		if err != nil {
			if sleepOnFailedToReceiveMessages(err) {
				continue
			}

			return err
		}

		//Here if we encounter an error, we don't want to stop processing, we want to return the error to the caller
		//via the SQS queue
		responseQueue, certificate, err := processNewCertificateRequestMessage(awsRegion, receipt, request)
		if err != nil {
			logger.WithError(err)
		}

		err = sendCertificateReply(awsRegion, roleArn, responseQueue, certificate, err)
		if err != nil {
			return err
		}

		err = aws_helpers.DeleteMessageFromQueue(awsRegion, roleArn, requestUrl, receipt)
		if err != nil {
			return err
		}

		logger.Info("DONE")
	}

	return nil
}

func waitForRequestMessage(awsRegion string, roleArn string, responseQueue string, timeout int) (receipt string, message string, error error) {
	receipt, message, err := aws_helpers.WaitForQueueMessage(awsRegion, roleArn, responseQueue, timeout)
	if err != nil {
		return "", "", err
	}

	return receipt, message, nil
}

func processNewCertificateRequestMessage(awsRegion string, receipt string, message string) (string, string, error) {

	request := CertificateRequest{}
	json.Unmarshal([]byte(message), &request)

	certificateAlreadyExists, err := indexContainsValidCertificate(request.Username)
	if err != nil {
		return "", "", err
	}

	if (!certificateAlreadyExists) {
		certificate, err := generateCertificate(request.Username)
		if err != nil {
			return request.ResponseQueue, "", err
		}

		return request.ResponseQueue, certificate, nil
	} else {
		var alreadyExistsError = fmt.Errorf("a valid certificate for %s already exists", request.Username)
		return request.ResponseQueue, "", errors.WithStackTrace(alreadyExistsError)
	}

}

func sendCertificateReply(awsRegion string, roleArn string, responseQueue string, certificate string, error error) error {

	responseMessage := &CertificateResponse{}
	responseMessage.Success = (error == nil)

	if responseMessage.Success {
		responseMessage.Body = certificate
	} else {
		responseMessage.ErrorMessage = error.Error()
	}

	requestJson, err := json.Marshal(responseMessage)
	if err != nil {
		return err
	}

	err = aws_helpers.SendMessageToQueue(awsRegion, roleArn, responseQueue, string(requestJson))
	if err != nil {
		return err
	}
	return nil
}

