# OpenVPN Host Examples

This folder shows an example of how to use the [openvpn-server module](/modules/openvpn-server) to launch the necessary 
components to serve as an OpenVPN host and to manage OpenVPN user accounts. 

## Quick start

To try these templates out you must have Terraform installed (minimum version: `0.8.0`):

1. Open `vars.tf`, set the environment variables specified at the top of the file, and fill in any other variables that
   don't have a default.
1. Run `terraform get`.
1. Run `terraform plan`.
1. If the plan looks good, run `terraform apply`.

## Why an OpenVPN Host?

Your team will need the ability to connect to resources hosted on EC2 Instances that you may not want to make accessible
to the entire Internet. 

Opening all these services directly is not advisable because then we have multiple servers and services that represent 
potential attack vectors into your environment. Instead, the best practice is to have a single server that's exposed to 
the public -- an "OpenVPN host" -- on which we can focus all our efforts for locking down. 

As a result, we place the OpenVPN host in the public subnet, and all other servers should be located in private subnets.
Once connected to the OpenVPN host, you can then connected to other services located on your private EC2 Instances.

## OpenVPN AMI

The OpenVPN host can run any reasonably secure Linux distro. In this example, we use an Ubuntu AMI built in the 
[packer example](/examples/packer).

## OpenVPN access

Once you have requested and received an OpenVPN Client Configuration (.ovpn) file using the 
[openvpn-admin](/modules/openvpn-admin) tool, you can import that into any supported OpenVPN client. 

Once connected to the VPN, your traffic is encrypted and "tunneled" and you are effectively "in the network". 
You can then connect to any other EC2 instance or service in the account, including those in the private subnets of the 
VPC.

## Known Limitations

When you first run the OpenVPN server it will take a long time, often 10 minutues or more until the first-time initialization 
of the public key infrastructure (PKI) necessary for running OpenVPN has initialized. You can tell that this process is 
complete once the `/etc/openvpn/openvpn-init-complete` file has been created.