# Backup PKI Assets Module

This module is used to backup the OpenVPN Public Key Infrastructure (PKI) to S3 on a server that has been installed using the [install-openvpn](../install-openvpn) module.

## How do you use this module?

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
