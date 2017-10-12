package app

import (
	"github.com/urfave/cli"
	"github.com/gruntwork-io/gruntwork-cli/logging"
	"fmt"
	"encoding/json"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/gruntwork-io/package-openvpn/modules/openvpn-admin/src/aws_helpers"
	"io/ioutil"
)

type CertificateRequest struct {
	Username      string
	ResponseQueue string
}

type CertificateResponse struct {
	Success      bool
	Body         string
	ErrorMessage string
}

func requestNewCertificate(cliContext *cli.Context) error {
	setLoggerLevel(cliContext)
	logger := logging.GetLogger(LOGGER_NAME)

	logger.Infof("Looking up AWS username")
	awsRegion, err := getAwsRegion(cliContext)
	if err != nil {
		return err
	}
	logger.Debugf("Using AWS Region: %s", awsRegion)

	username, err := getUsername(cliContext, true)
	if err != nil {
		return err
	}
	logger.Debugf("Using Username: %s", username)

	logger.Infof("Looking up SQS queue")
	requestUrl, err := getRequestUrl(cliContext)
	if err != nil {
		return err
	}
	logger.Debugf("Using Request URL: %s", requestUrl)

	timeout, err := getTimeout(cliContext)
	if err != nil {
		return err
	}

	//Create a new response queue
	responseQueue, err := createResponseQueue(awsRegion)
	if err != nil {
		return err
	}
	defer deleteResponseQueue(awsRegion, responseQueue)

	logger.Infof("Submitting request for new certificate to %s", responseQueue)
	//Put a request for a new certificate on the requestQueue
	err = sendRequest(awsRegion, requestUrl, username, responseQueue)
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
	err = processNewCertificateResponse(awsRegion, responseQueue, receipt, response, username)
	if err != nil {
		return err
	}

	logger.Info("DONE")
	return nil
}

func sendRequest(awsRegion string, requestUrl string, username string, responseQueue string) error {
	req := &CertificateRequest{
		Username: username,
		ResponseQueue:responseQueue,
	}
	requestJson, _ := json.Marshal(req)

	err := aws_helpers.SendMessageToQueue(awsRegion, requestUrl, string(requestJson))
	if err != nil {
		return err
	}
	return nil
}

func processNewCertificateResponse(awsRegion string, resonseQueue string, receipt string, message string, username string) error {
	response := CertificateResponse{}
	json.Unmarshal([]byte(message), &response)

	if (!response.Success) {
		aws_helpers.DeleteMessageFromQueue(awsRegion, resonseQueue, receipt)
		return errors.WithStackTrace(fmt.Errorf(response.ErrorMessage))
	} else {
		err := createOvpnFile(username, response.Body)
		if err != nil {
			return err
		}
		aws_helpers.DeleteMessageFromQueue(awsRegion, resonseQueue, receipt)
	}

	return nil
}

func createOvpnFile(username string, contents string) (error) {
	filename := "./" + username + ".ovpn"

	logger := logging.GetLogger(LOGGER_NAME)
	logger.Info(fmt.Sprintf("Creating OpenVpn configuration file %s", filename))

	err := ioutil.WriteFile(filename, []byte(contents), 0644)
	if err != nil {
		return err
	}

	return nil
}