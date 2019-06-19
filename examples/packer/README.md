# Open VPN example AMI

This folder contains an example [Packer](https://www.packer.io/) template for building an AMI (Amazon Machine Image) containing  the OpenVPN server.

## Pre-requisites:

In order to build this AMI you will need to provide some input variables. There are several variables but the most important ones are:

* In which AWS region should Packer build your AMI
* Where should Packer look for a copy of the [openvpn-admin](/modules/openvpn-admin) binary that you will use to manage your VPN certificates. If you haven't built this before, take a look at it's documentation for steps on how to build it. **Remember:** When building `openvpn-admin` for use in this packer template, keep in mind the OS and architecture where this will be _deployed_ and not the OS/architecture of the machine that's building `openvpn-admin`. E.g., if you're firing up an EC2 Instance that runs Linux, you'll need to build the binary for Linux, even if you happen to be running the build on a Mac.

All variables below:

| Variable name               | Description                                                  | Default Value                 |
| --------------------------- | ------------------------------------------------------------ | ----------------------------- |
| aws_region                  | Tells Packer in which AWS region to build your AMI           | `us-east-1`                   |
| github_oauth_token          | Your github OAuth token.                                     | `env.GITHUB_OAUTH_TOKEN`      |
| openvpn_admin_binary        | Where should Packer look for a copy of the `openvpn-admin` binary that you will use to manage the VPN certificates on your VPN server. See: [openvpn-admin](/modules/openvpn-admin) for more info. | `/examples/bin/openvpn-admin` |
| gruntwork_installer_version | What version of [Gruntwork Installer](https://github.com/gruntwork-io/gruntwork-installer) to use | `v0.0.20`                     |
| bash_commons_version        | What version of [bash-commons](https://github.com/gruntwork-io/bash-commons) to use | `v0.0.6`                      |

## Building the packer template

Below is an example of the command you could run to build this packer template.

```bash
packer build \
	-var aws_region=us-east-1 \
	-var openvpn_admin_binary=../examples/bin/openvpn-admin \
	-only=ubuntu-16-build \
	../examples/packer/build.json
```

**Notes**

- `-only` flag allows you to execute one specific builder with the given name.  This parameter can be ommitted if the packer template has only one builder defined in it. This example has just one builder, but we're including the parameter for reference.
- `openvpn_admin_source` in the example above is pointing to a sample location. Unless you execute the [automated test](/test/openvpn_test.go), which will actually build the artifact for you and place it into that folder, you will have to provide your own path where that binary is located.
