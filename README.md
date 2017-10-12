
# Open VPN Package Infrastructure Package

This repo contains modules for running a production-ready OpenVPN server and managing OpenVPN user accounts. The modules are:

* [init-openvpn](/modules/init-openvpn) - initializes the public key infrastructure (PKI) via OpenSSL for use by OpenVPN. Designed to be run via user-data on boot 
* [install-openvpn](/modules/install-openvpn) - Scripts to install the OpenVPN image in a packer-generated AMI
* [openvpn-admin](/modules/openvpn-admin) - A command-line utility to request and revoke certificates and to process those requests
* [openvpn-server](/modules/openvpn-server) - Terraform templates that deploy OpenVPN
* [start-openvpn-admin](/modules/start-openvpn-admin) - Scripts to start [openvpn-admin](/modules/openvpn-admin) on the [openvpn-server](/modules/openvpn-server) in order to process certificate requests and revocations

## Architecture Overview

#### Server
The [openvpn-server](./modules/openvpn-server) module will deploy a single server into an Auto Scaling Group (ASG) for redundancy
purposes. Should the server fail, a new server will be automatically provisioned by the ASG. Upon initial boot, the 
[init-openvpn](./modules/init-openvpn) module will establish a new public key infrastructure (PKI), including
a Certificate Authority (CA), server certificate and a certificate revocation list. These assets will then be backed up  
to an S3 bucket and encrypted for protection should a server failure occur. 

In a failure scenario, when a replacement server is started by the ASG, the PKI will be automatically restored from the 
S3 bucket ensuring the previously-issued client certificates will continue to function.

#### Client Certificate Requests
Users who are members of the proper IAM group will use the [openvpn-admin](./modules/openvpn-admin) utility to request
a new certificate. Behind the scenes, this certificate request will be sent to the server via an SQS queue, will be signed by
the server and an OpenVPN client configuration file (.ovpn) with the certificates embedded will be written to disk on the 
requestor's workstation. This `.ovpn` file can then be imported into any number of popular OpenVPN clients.

#### Client Certificate Revocations
Users who are members of the proper IAM group will be allowed to use the same [openvpn-admin](./modules/openvpn-admin) 
utility to revoke an existing user's certificate. Behind the scenes, the revocation requests will be sent to the server
via an SQS queue, the certificate will be revoked and a confirmation will be sent to back to the requestor's workstation.

#### Usage with multiple AWS accounts
If your IAM users are defined in one AWS account (e.g., security account) and the OpenVPN server is deployed in another
account (e.g., the dev or prod account), then in order for users to be able to request or revoke OpenVPN certificates, 
they will need access to the SQS queues in the account with the OpenVPN server. When deploying the [openvpn-server 
module](/modules/openvpn-server), you can specify the ARNs of the AWS account where IAM users are defined using the
`external_account_arns` parameter, and the module will create two IAM roles—one for users and one for admins—that can be
assumed by users in those accounts to get access to the SQS queues. See the [how to switch between accounts
documentation](https://github.com/gruntwork-io/module-security/tree/master/modules/cross-account-iam-roles#how-to-switch-between-accounts)
for instructions on assuming IAM roles in other AWS accounts.  
  

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

1. Convert to CIDR format for parameters
