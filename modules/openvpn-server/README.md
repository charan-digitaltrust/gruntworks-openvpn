# OpenVPN Server Module

This module makes it easy to deploy an OpenVPN server in an auto-scaling group (size 1) for fault tolerance --along with the all the resources it typically needs:

1. The Auto-Scaling Group.
1. An EC2 Instance
1. An Elastic IP (EIP) address.
1. IAM Role and IAM instance profile.
1. Simple Queuing Services (SQS) Queues
1. An S3 Bucket for certificate backups
1. Security groups.

## How do you use this module?

* See the [root README](/README.md) for instructions on using Terraform modules.
* See the [examples](/examples) folder for example usage.
* See [vars.tf](./vars.tf) for all the variables you can set on this module.

## How do I access the server?

This module include several [Terraform outputs](https://www.terraform.io/intro/getting-started/outputs.html),
including:

1. `public_ip`: The public IP address of the server (via its EIP)

## How do I add custom security group rules?

One of the other important outputs of this module is the `security_group_id`, which is the id of the server's Security
Group. You can add custom rules to this Security Group using the `aws_security_group_rule` resource:

```hcl
module "openvpn" {
  source = "git::git@github.com:gruntwork-io/package-openvpn.git//modules/openvpn-server?ref=v0.0.40"

  # (... options omitted...)
}

# Custom rule to allow inbound HTTPS traffic from anywhere
resource "aws_security_group_rule" "allow_inbound_https_all" {
  type = "ingress"
  from_port = 443
  to_port = 443
  protocol = "tcp"
  cidr_blocks = ["0.0.0.0/0"]
  security_group_id = "${module.openvpn.security_group_id}"
}
```

## How do I add a custom IAM policy?

This module creates an IAM role for your EC2 instance and exports the id of that role as the output `iam_role_id`. You
can attach custom policies to this IAM role using the `aws_iam_policy_attachment` resource:

```hcl
module "openvpn" {
  source = "git::git@github.com:gruntwork-io/package-openvpn.git//modules/openvpn-server?ref=v0.0.40"

  # (... options omitted...)
}

resource "aws_iam_policy" "my_custom_policy" {
  name = "my-custom-policy"
  policy = " (... omitted ...) "
}

resource "aws_iam_policy_attachment" "attachment" {
  name = "example-attachment"
  roles = ["${module.openvpn.iam_role_id}"]
  policy_arn = "${aws_iam_policy.my_custom_policy.arn}"
}
```

## What if I want to enable MFA?

The scripts [init-openvpn](../init-openvpn) and [install-openvpn](../install-openvpn) support setting up the
[duo_openvpn](https://github.com/duosecurity/duo_openvpn) plugin for 2FA authentication. To enable the duo plugin, you
need to:

1. Build an AMI that has the `duo_openvpn` plugin installed. You can use `install-openvpn` to install the plugin
   alongside openvpn by passing in the argument `--duo-version`. For example:

     sudo /usr/local/bin/install-openvpn --duo-version 2.2

1. In the `user_data` script for the server, pass in the duo keys to `init-openvpn` using the arguments `--duo-ikey`,
   `--duo-skey`, and `--duo-host` to configure the integration key, secret key, and API hostname respectively.

See the [packer-duo](../examples/packer-duo) and [openvpn-host-duo](../examples/openvpn-host-duo) examples for an
example configuration to deploy the OpenVPN server with Duo enabled.
