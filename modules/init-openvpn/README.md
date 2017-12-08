# Init OpenVPN Module

This module is used to initialize the OpenVPN server, its Public Key Infrastructure (PKI), Certificate Authority
(CA) and configuration on a server that has been installed using the [install-openvpn](../install-openvpn) module.

## How do you use this module?

#### Example

See the [example](/examples/openvpn-host) for an example of how to use this module.

#### Installation

```
gruntwork-install --module-name init-openvpn --tag v0.4.0 --repo https://github.com/gruntwork-io/package-openvpn
```

#### Configuration Options
You can configure several options to control the behavior of OpenVPN. 

|Option|Description|Required|Default|
|-------------------------|---|---|-------------|
|--s3-bucket-name|The name of an S3 bucket that will be used to backup the PKI|Required
|--kms-key-id|The id of a KMS key that will used to encrypt/decrypt the PKI when stored in S3|Required
|--email|The e-mail address of the administrator. Used in the CA configuration|Required
|--org-unit|The name of the unit, department, or scope within your organization for which this CA certificate will be used|Required
|--org|The name of your organization (e.g. Gruntwork)|Required
|--locality|The locality name (e.g. city or town name) where your organization is located|Required
|--state|The state or province name where your organization is located. Use the full, unabbreviated name. E.g. New Jersey|Required
|--country|The two-letter country name where your organization is located (see https://www.digicert.com/ssl-certificate-country-codes.htm)|Required
|--vpn-subnet|The subnet the vpn clients will be assigned addresses from, in subnet mask format. Eg, "10.1.14.0 255.255.255.0"|Required
|--vpn-route|Routes to subnets that will be protected by the VPN and will be pushed to the VPN clients, in [subnet] [mask] format. Eg, "10.100.0.0 255.255.255.0". Can be specified multiple times.|Required
|--key-size|The key size (in bits) for server and client certificates|Optional|4096
|--ca-expiration-days|The number of days the CA root certificate will be valid for|Optional|3650 (10 years)
|--cert-expiration-days|The number of days a server or user certificate issued by the CA will be valid for|Optional|3650 (10 years)
|--crl-expiration-days|The number of days the CA Certificate Revocation List (CRL) will be valid for|Optional|3650 (10 years)


#### Configure the OpenVPN Package on your EC2 Instances

In order for the EC2 Instance to run OpenVPN sucessfully, it needs certain data from the EC2 instance.

When your EC2 Instances are booting up, they should run the `init-openvpn` script, which will configure
OpenVPN on your instance. 

The best way to run a script during boot is to put it in [User
Data](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/user-data.html#user-data-shell-scripts). Here's an example:

```bash
#!/bin/bash
echo 'Initializing PKI and Copying OpenVPN config into place...'
sudo init-openvpn  \
 --country "US"  \
 --state "NJ"  \
 --locality "Marlboro"  \
 --org "Acme"  \
 --org-unit "OpenVPN"  \
 --email "itsupport@acme.none"  \
 --s3-bucket-name "acme-openvpn-backups"  \
 --kms-key-id "fd805ce5-2d70-4144-9370-2d9d2ed265fb"  \
 --key-size "4096" \
 --ca-expiration-days "3650" \
 --cert-expiration-days "3650" \
 --crl-expiration-days "3650" \
 --vpn-subnet "10.1.14.0 255.255.255.0" \
 --vpn-route "10.100.0.0 255.255.0.0" \ 
 --vpn-route "10.101.0.0 255.255.0.0" \
 --vpn-route "10.102.0.0 255.255.0.0"
```
#### Note
The initial generation of PKI is very CPU intensive and can take a long time (30+ minutes), especially on baseline/burst
type instances such as the `t2` family. See [here](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/t2-instances.html#t2-instances-cpu-credits)
for additional information.