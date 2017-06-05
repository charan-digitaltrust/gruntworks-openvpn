package app

import (
	"os/exec"
	"strings"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/gruntwork-io/gruntwork-cli/logging"
	"fmt"
	"regexp"
	"github.com/gruntwork-io/gruntwork-cli/files"
)

type certificatePartData struct {
	IpAddress       string
	CaCertificate   string
	UserCertificate string
	UserKey         string
	Error           error
}

func generateCertificate(username string) (string, error) {
	command := exec.Command("./generate-wrapper.sh", username)
	command.Dir = "/etc/openvpn-ca"

	_, err := command.CombinedOutput()
	if err != nil {
		return "", errors.WithStackTrace(err)
	}

	content, err := generateCertificateTemplate(username)
	if err != nil {
		return "", errors.WithStackTrace(err)
	}

	return content, nil
}

func generateCertificateTemplate(username string) (string, error) {
	template, err := readTemplateFile()
	if err != nil {
		return "", err
	}

	data := getCertificatePartData(username)
	if data.Error != nil {
		return "", data.Error
	}

	template = strings.Replace(template, "__SERVER_ADDRESS__", data.IpAddress, -1)
	template = strings.Replace(template, "__CA_CERTIFICATE__", data.CaCertificate, -1)
	template = strings.Replace(template, "__CLIENT_CERTIFICATE__", data.UserCertificate, -1)
	template = strings.Replace(template, "__CLIENT_KEY__", data.UserKey, -1)

	return template, nil
}

func getCertificatePartData(username string) certificatePartData {
	ipAddress, err := getIpAddress()
	if err != nil {
		return certificatePartData{Error:err, }
	}

	caCert, err := readCaCert()
	if err != nil {
		return certificatePartData{Error:err, }
	}

	userCert, err := readUserCert(username)
	if err != nil {
		return certificatePartData{Error:err, }
	}

	userKey, err := readUserKey(username)
	if err != nil {
		return certificatePartData{Error:err, }
	}

	return certificatePartData{
		IpAddress:ipAddress,
		CaCertificate:caCert,
		UserCertificate:userCert,
		UserKey: userKey,
	}
}

func revokeCertificate(username string) (error) {
	logger := logging.GetLogger(LOGGER_NAME)
	logger.Debugf("Running revoke script ./revoke-wrapper.sh %s", username)
	command := exec.Command("./revoke-wrapper.sh", username)
	command.Dir = "/etc/openvpn-ca"

	output, err := command.CombinedOutput()
	if err != nil {
		return errors.WithStackTrace(err)
	}

	//fmt.Printf("%s\n", output)
	match, err := regexp.MatchString("Already revoked", string(output))
	if err != nil {
		return err
	}

	if match {
		return fmt.Errorf("Certificate for %s already revoked", username)
	}

	return nil
}

func readTemplateFile() (string, error) {
	return files.ReadFileAsString("/etc/openvpn/openvpn-client.ovpn")
}

func readCaCert() (string, error) {
	return files.ReadFileAsString("/etc/openvpn/ca.crt")
}

func readUserCert(username string) (string, error) {
	return files.ReadFileAsString("/etc/openvpn/" + username + ".crt")
}

func readUserKey(username string) (string, error) {
	return files.ReadFileAsString("/etc/openvpn/" + username + ".key")
}
