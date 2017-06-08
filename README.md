
# Open VPN Package Infrastructure Package

This repo contains modules for running a production-ready OpenVPN server and managing OpenVPN user accounts. The modules are:

* [init-openvpn](/modules/init-openvpn) - initializes the public key infrastructure (PKI) via OpenSSL for use by OpenVPN. Designed to be run via user-data on boot 
* [install-openvpn](/modules/install-openvpn) - Scripts to install the OpenVPN image in a packer-generated AMI
* [openvpn-admin](/modules/openvpn-admin) - A command-line utility to request and revoke certificates and to process those requests
* [openvpn-server](/modules/openvpn-server) - Terraform templates that deploy OpenVPN
* [start-openvpn-admin](/modules/start-openvpn-admin) - Scripts to start [openvpn-admin](/modules/openvpn-admin) on the [openvpn-server](/modules/openvpn-server) in order to process certificate requests and revocations

## What is a module?

At [Gruntwork](http://www.gruntwork.io), we've taken the thousands of hours we spent building infrastructure on AWS and
condensed all that experience and code into pre-built **packages** or **modules**. Each module is a battle-tested,
best-practices definition of a piece of infrastructure, such as a VPC, ECS cluster, or an Auto Scaling Group. Modules
are versioned using [Semantic Versioning](http://semver.org/) to allow Gruntwork clients to keep up to date with the
latest infrastructure best practices in a systematic way.

## How do you use a module?

Most of our modules contain either:

1. [Terraform](https://www.terraform.io/) code
1. Scripts & binaries

#### Using a Terraform Module

To use a module in your Terraform templates, create a `module` resource and set its `source` field to the Git URL of
this repo. You should also set the `ref` parameter so you're fixed to a specific version of this repo, as the `master`
branch may have backwards incompatible changes (see [module
sources](https://www.terraform.io/docs/modules/sources.html)).

For example, to use `v1.0.0` of the openvpn module, you would add the following:

```hcl
module "openvpn-server" {
  source = "git::git@github.com:gruntwork-io/module-openvpn.git//modules/openvpn-server?ref=v1.0.0"

  // set the parameters for the OpenVPN module
}
```

*Note: the double slash (`//`) is intentional and required. It's part of Terraform's Git syntax (see [module
sources](https://www.terraform.io/docs/modules/sources.html)).*

See the module's documentation and `vars.tf` file for all the parameters you can set. Run `terraform get -update` to
pull the latest version of this module from this repo before runnin gthe standard  `terraform plan` and
`terraform apply` commands.

#### Using scripts & binaries

You can install the scripts and binaries in the `modules` folder of any repo using the [Gruntwork
Installer](https://github.com/gruntwork-io/gruntwork-installer). For example, if the scripts you want to install are
in the `modules/mongodb-scripts` folder of the https://github.com/gruntwork-io/package-mongodb repo, you could install them
as follows:

```bash
gruntwork-install --module-name "init-openvpn" --repo "https://github.com/gruntwork-io/package-openvpn" --tag "0.0.1"
```

See the docs for each script & binary for detailed instructions on how to use them.

## Developing a module

#### Versioning

We are following the principles of [Semantic Versioning](http://semver.org/). During initial development, the major
version is to 0 (e.g., `0.x.y`), which indicates the code does not yet have a stable API. Once we hit `1.0.0`, we will
follow these rules:

1. Increment the patch version for backwards-compatible bug fixes (e.g., `v1.0.8 -> v1.0.9`).
2. Increment the minor version for new features that are backwards-compatible (e.g., `v1.0.8 -> v1.1.0`).
3. Increment the major version for any backwards-incompatible changes (e.g. `v1.0.8 -> v2.0.0`).

The version is defined using Git tags.  Use GitHub to create a release, which will have the effect of adding a git tag.

#### Examples

See the [examples](/examples) folder for sample code to build the openvpn-admin binary, a packer template to build an AMI and Terraform code to launch everything necessary to run OpenVPN in your AWS environment.

#### Tests

See the [test](/test) folder for details.

## License

Please see [LICENSE.txt](/LICENSE.txt) for details on how the code in this repo is licensed.

## ToDo

1. Update documentation for sub-modules
2. Convert to CIDR format for parameters