# Install OpenVPN Module

This module is used to install the OpenVPN package and related template files onto a server. It is expected that
the [init-openvpn](../init-openvpn) module will be run on the server during boot to configure the OpenVPN server installed by this
package.

## How do you use this module?

_**Note:** This module uses explicitly `easy-rsa v2.2.2`. There is planned future work to migrate over to a more recent version. More details can be found on [this issue](https://github.com/gruntwork-io/terraform-aws-openvpn/issues/108)._

#### Example

See the [example](/examples/openvpn-host) for an example of how to use this module.

#### Installation

```bash
#!/bin/bash
sudo gruntwork-install --module-name install-openvpn --tag v0.4.0 --repo https://github.com/gruntwork-io/terraform-aws-openvpn
sudo /usr/local/bin/install-openvpn
```

##### Install the OpenVPN Package on your AMI

In order for the [openvpn-server](../openvpn-server) module to work properly, you have to build an AMI with the
OpenVPN package installed using this module. Please see [example](/examples/packer) for a sample packer build script.