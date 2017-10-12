package app

import (
	"regexp"
	"fmt"
	"github.com/gruntwork-io/package-openvpn/modules/openvpn-admin/src/aws_helpers"
	"net/http"
	"strings"
	"time"
	"github.com/gruntwork-io/gruntwork-cli/logging"
	"github.com/gruntwork-io/gruntwork-cli/files"
	"io/ioutil"
)

func createResponseQueue(awsRegion string) (string, error) {
	queueUrl, err := aws_helpers.CreateRandomQueue(awsRegion, "openvpn-response")
	if err != nil {
		return "", err
	}
	return queueUrl, nil
}

func deleteResponseQueue(awsRegion string, responseQueue string) (error) {
	err := aws_helpers.DeleteQueue(awsRegion, responseQueue)
	if err != nil {
		return err
	}
	return nil
}

func waitForMessage(awsRegion string, queue string, timeout int) (string, string, error) {
	receipt, message, err := aws_helpers.WaitForQueueMessage(awsRegion, queue, timeout)
	if err != nil {
		return "", "", err
	}

	return receipt, message, nil
}

func indexContainsValidCertificate(username string) (bool, error) {
	indexFile, err := files.ReadFileAsString("/etc/openvpn/index.txt")
	if err != nil {
		return false, err
	}

	pattern := fmt.Sprintf("V\\s+.*(CN=%s)", username)
	return regexp.MatchString(pattern, indexFile)
}

func getIpAddress() (string, error) {
	var client http.Client
	resp, err := client.Get("http://169.254.169.254/latest/meta-data/public-ipv4")
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	var ipaddress string
	if resp.StatusCode == 200 {
		// OK
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		ipaddress = string(bodyBytes)
	}
	return ipaddress, nil
}

func sleepOnFailedToReceiveMessages(err error) bool {
	logger := logging.GetLogger(LOGGER_NAME)
	if strings.Contains(err.Error(), "Failed to receive messages") {
		logger.Warn(fmt.Sprintf("%s, sleeping for 30 seconds before retrying", err.Error()))
		time.Sleep(time.Second * 30)
		return true
	}
	return false
}