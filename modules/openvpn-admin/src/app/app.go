package app

import (
	"github.com/urfave/cli"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"fmt"
)

const LOGGER_NAME = "openvpn-admin"
const OPTION_USERNAME = "username"
const OPTION_TIMEOUT = "timeout"
const OPTION_AWS_REGION = "aws-region"
const OPTION_DEBUG = "debug"
const OPTION_REQUEST_URL = "request-url"
const OPTION_REVOKE_URL = "revoke-url"

func CreateApp(version string) *cli.App {
	app := cli.NewApp()

	app.Name = "openvpn-admin"
	app.Author = "Gruntwork <www.gruntwork.io>"
	app.UsageText = "openvpn-admin <COMMAND> [OPTIONS]"
	app.Version = version
	app.Usage = `openvpn-admin is a command-line tool that makes it easy to request or revoke certificates for use
	with an OpenVPN server installed with the Gruntwork package-openvpn module.`

	awsRegionFlag := cli.StringFlag{
		Name: OPTION_AWS_REGION,
		Usage: "The AWS region where your customer master key (CMK) is defined (e.g. us-east-1).",
		EnvVar: "AWS_DEFAULT_REGION",
	}

	usernameFlag := cli.StringFlag{
		Name: OPTION_USERNAME,
		Usage: "The username that the certificate is being requested for. Defaults to current IAM username when requesting a cert; required when revoking a cert.",
	}

	timeoutFlag := cli.IntFlag{
		Name: OPTION_TIMEOUT,
		Usage: "The maximum number of seconds to wait for a response from the OpenVPN server. Defaults to 300",
		Value: 300,
	}

	requestUrlFlag := cli.StringFlag{
		Name: OPTION_REQUEST_URL,
		Usage: "The SQS url of the certificate request queue. Optional.",
	}

	revokeUrlFlag := cli.StringFlag{
		Name: OPTION_REVOKE_URL,
		Usage: "The SQS url of the certificate revocation queue. Optional.",
	}

	debugFlag := cli.BoolFlag{
		Name: OPTION_DEBUG,
		Usage: "Whether debug logging should be enabled",
		EnvVar: "OPENVPN_ADMIN_DEBUG",
	}

	app.Commands = []cli.Command{
		{
			Name: "request",
			Usage: "Request a new certificate for a user with OpenVPN",
			Action: errors.WithPanicHandling(requestNewCertificate),
			Flags: []cli.Flag{debugFlag, requestUrlFlag, revokeUrlFlag, usernameFlag, timeoutFlag, awsRegionFlag},
		},
		{
			Name: "revoke",
			Usage: "Revoke an existing OpenVPN certificate for a user",
			Action: errors.WithPanicHandling(requestCertificateRevocation),
			Flags: []cli.Flag{debugFlag, requestUrlFlag, revokeUrlFlag, usernameFlag, awsRegionFlag, timeoutFlag},
		},
		{
			Name: "process-requests",
			Usage: "Listen for certificate requests and revocations and process those requests",
			Action: errors.WithPanicHandling(processNewCertificateRequests),
			Flags: []cli.Flag{debugFlag, requestUrlFlag, revokeUrlFlag, usernameFlag, awsRegionFlag, timeoutFlag},
		},
		{
			Name: "process-revokes",
			Usage: "Listen for certificate revocations and process those requests",
			Action: errors.WithPanicHandling(processCertificateRevocationRequests),
			Flags: []cli.Flag{debugFlag, requestUrlFlag, revokeUrlFlag, usernameFlag, awsRegionFlag, timeoutFlag},
		},
	}

	app.CommandNotFound = commandNotFound

	return app
}

func commandNotFound(cliContext *cli.Context, command string) {
	fmt.Fprintf(cliContext.App.Writer, "Error: unrecognized command '%s'", command)
}


// Custom errors
var MissingUsername = fmt.Errorf("--%s cannot be empty", OPTION_USERNAME)
var MissingAwsRegion = fmt.Errorf("--%s cannot be empty", OPTION_AWS_REGION)
var MissingRequestUrl = fmt.Errorf("--%s cannot be empty", OPTION_REQUEST_URL)
var MissingRevokeUrl = fmt.Errorf("--%s cannot be empty", OPTION_REVOKE_URL)
