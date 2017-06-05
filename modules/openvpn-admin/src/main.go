package main

import (
	"github.com/gruntwork-io/gruntwork-cli/entrypoint"
	"github.com/gruntwork-io/package-openvpn/modules/openvpn-admin/src/app"
)

// This variable is set at build time using -ldflags parameters. For example, we typically set this flag in circle.yml
// to the latest Git tag when building our Go apps:
//
// build-go-binaries --app-name my-app --dest-path bin --ld-flags "-X main.VERSION=$CIRCLE_TAG"
//
// For more info, see: http://stackoverflow.com/a/11355611/483528
var VERSION string

func main() {
	openVpnAdminApp := app.CreateApp(VERSION)
	entrypoint.RunApp(openVpnAdminApp)
}