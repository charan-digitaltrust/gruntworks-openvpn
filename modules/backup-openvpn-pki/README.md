# Backup PKI Assets Module

This module is used to backup the OpenVPN Public Key Infrastructure (PKI) to S3 on a server that has been installed using 
the [install-openvpn](../install-openvpn) module. 

The PKI is the set of certificates used to verify the server and users' identities for VPN authentication purposes. This
normally lives on the OpenVPN server in the `/etc/openvpn-ca` and `/etc/openvpn` directories. If we didn't back these files
up, we would have to reissue client certificates if the OpenVPN server ever needed to be rebuilt. 

## How do you use this module?

This module is used by the `init-openvpn` module to backup the PKI on initial installation. The `init-openvpn` module
will also install a `cron` job to automatically backup the PKI on an hourly basis.

#### Example

See the [example](/examples/openvpn-host) for an example of how to use this module.

#### Installation

```
gruntwork-install --module-name backup-openvpn-pki --tag v0.4.1 --repo https://github.com/gruntwork-io/package-openvpn
```

#### Configuration Options

|Option|Description|Required|Default|
|-------------------------|---|---|-------------|
|--s3-bucket-name|The name of an S3 bucket that will be used to backup the PKI|Required
|--kms-key-id|The id of a KMS key that will used to encrypt/decrypt the PKI when stored in S3|Required
