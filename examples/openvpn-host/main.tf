# ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
# LAUNCH THE OPENVPN HOST
# The OpenVPN host is the sole point of entry to the network. This way, we can make all other servers inaccessible from
# the public Internet and focus our efforts on locking down the OpenVPN host.
# ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

# ---------------------------------------------------------------------------------------------------------------------
# CONFIGURE OUR AWS CONNECTION
# ---------------------------------------------------------------------------------------------------------------------

provider "aws" {
  # The AWS region in which all resources will be created
  region = "${var.aws_region}"
}

resource "aws_kms_key" "backups" {
  description = "OpenVPN Backup Key"
}


# ---------------------------------------------------------------------------------------------------------------------
# SETUP DATA STRUCTURES
# ---------------------------------------------------------------------------------------------------------------------
data "aws_vpc" "default" {
  default = true
}

data "aws_region" "current" {
  current = true
}

data "aws_availability_zones" "available" {}

data "aws_subnet" "default" {
  vpc_id = "${data.aws_vpc.default.id}"
  availability_zone = "${data.aws_availability_zones.available.names[0]}"
}

data "template_file" "user_data" {
  template = "${file("${path.module}/user-data/user-data.sh")}"

  vars {
    backup_bucket_name = "${module.openvpn.backup_bucket_name}"
    key_size = "${module.openvpn.key_size}"
    ca_expiration_days = "${module.openvpn.openssl_ca_expiration_days}"
    cert_expiration_days = "${module.openvpn.openssl_certificate_expiration_days}"
    ca_country = "${module.openvpn.ca_country}"
    ca_state = "${module.openvpn.ca_state}"
    ca_locality = "${module.openvpn.ca_locality}"
    ca_org = "${module.openvpn.ca_org}"
    ca_org_unit = "${module.openvpn.ca_org_unit}"
    ca_email = "${module.openvpn.ca_email}"
    eip_id = "${module.openvpn.elastic_ip}"
    request_queue_url = "${module.openvpn.client_request_queue}"
    revocation_queue_url = "${module.openvpn.client_revocation_queue}"
    queue_region = "${data.aws_region.current.name}"
    vpn_subnet = "${cidrhost(data.aws_subnet.default.cidr_block,0)} ${cidrnetmask(data.aws_subnet.default.cidr_block)}"
    routes = "${chomp(join(" ", formatlist("--vpn-route \"%s\" ", module.openvpn.vpn_routes)))}"
  }
}

# ---------------------------------------------------------------------------------------------------------------------
# LAUNCH THE OPENVPN HOST
# ---------------------------------------------------------------------------------------------------------------------
module "openvpn" {
  # When using these modules in your own templates, you will need to use a Git URL with a ref attribute that pins you
  # to a specific version of the modules, such as the following example:
  # source = "git::git@github.com:gruntwork-io/module-openvpn.git//modules/openvpn-server?ref=v1.0.0"
  source = "../../modules/openvpn-server"

  name = "${var.name}"
  instance_type = "c4.large"
  ami = "${var.ami}"
  keypair_name = "${var.keypair_name}"
  user_data = "${data.template_file.user_data.rendered}"

  # Since s3 bucket names are globally unique, create a random suffix so multiple customers'
  # examples can work with just terraform apply and no need to change default vaules
  backup_bucket_name = "${var.backup_bucket_name}-${uuid()}"

  request_queue_name = "${var.request_queue_name}"
  revocation_queue_name = "${var.revocation_queue_name}"
  kms_key_arn = "${aws_kms_key.backups.arn}"
  vpc_id = "${data.aws_vpc.default.id}"
  subnet_id = "${data.aws_subnet.default.id}"

  #WARNING: This should be set to 4096 (default) for production, but this is much faster for test/dev
  openssl_key_size = "2048"

  #WARNING: Only allow SSH from everywhere for test/dev, never in production
  allow_ssh_from_cidr = true
  allow_ssh_from_cidr_list = [
    "0.0.0.0/0"
  ]

  #OpenVPN/CA Specific variable
  ca_state = "NJ"
  ca_country = "US"
  ca_org_unit = "OpenVPN"
  ca_email = "support@gruntwork.io"
  ca_locality = "Marlboro"
  ca_org = "Gruntwork"
  vpn_subnet = "192.168.99.0 255.255.255.0"
  vpn_routes = [
    #add the vpc's supernet
    "${cidrhost(data.aws_vpc.default.cidr_block,0)} ${cidrnetmask(data.aws_vpc.default.cidr_block)}"
  ]

  #WARNING: Only set this to true for testing/dev, never in production
  backup_bucket_force_destroy = "true"
}
