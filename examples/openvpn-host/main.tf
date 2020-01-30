# ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
# LAUNCH THE OPENVPN HOST
# The OpenVPN host is the sole point of entry to the network. This way, we can make all other servers inaccessible from
# the public Internet and focus our efforts on locking down the OpenVPN host.
# ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

terraform {
  required_version = ">= 0.12"
}

# ---------------------------------------------------------------------------------------------------------------------
# CONFIGURE OUR AWS CONNECTION
# ---------------------------------------------------------------------------------------------------------------------

provider "aws" {
  # The AWS region in which all resources will be created
  region = var.aws_region
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

data "aws_region" "current" {}

data "aws_availability_zones" "available" {}

data "aws_subnet" "default" {
  vpc_id            = data.aws_vpc.default.id
  availability_zone = data.aws_availability_zones.available.names[0]
}

data "template_file" "user_data" {
  template = file("${path.module}/user-data/user-data.sh")

  vars = {
    backup_bucket_name = module.openvpn.backup_bucket_name
    kms_key_id         = aws_kms_key.backups.id
    #WARNING: This should be set to 4096 (default) for production, but this is much faster for test/dev
    key_size             = 2048
    ca_expiration_days   = 3650
    cert_expiration_days = 3650
    ca_country           = "US"
    ca_state             = "NJ"
    ca_locality          = "Marlboro"
    ca_org               = "Gruntwork"
    ca_org_unit          = "OpenVPN"
    ca_email             = "support@gruntwork.io"
    eip_id               = module.openvpn.elastic_ip
    request_queue_url    = module.openvpn.client_request_queue
    revocation_queue_url = module.openvpn.client_revocation_queue
    queue_region         = data.aws_region.current.name
    search_domain        = "foo.bar.com"
    vpn_subnet           = "192.168.99.0 255.255.255.0"
    routes = chomp(
      join(
        " ",
        formatlist(
          "--vpn-route \"%s\" ",
          ["${cidrhost(data.aws_vpc.default.cidr_block, 0)} ${cidrnetmask(data.aws_vpc.default.cidr_block)}"],
        ),
      ),
    )
  }
}

# ---------------------------------------------------------------------------------------------------------------------
# CREATE REQUIRED VPC ENDPOINTS
# ---------------------------------------------------------------------------------------------------------------------

resource "aws_vpc_endpoint" "s3" {
  count        = var.allow_all_outbound_traffic ? 0 : 1
  vpc_id       = data.aws_vpc.default.id
  service_name = "com.amazonaws.${var.aws_region}.s3"
}

resource "aws_vpc_endpoint" "sqs" {
  count              = var.allow_all_outbound_traffic ? 0 : 1
  vpc_id             = data.aws_vpc.default.id
  service_name       = "com.amazonaws.${var.aws_region}.sqs"
  vpc_endpoint_type  = "Interface"
  security_group_ids = aws_security_group.sqs_endpoint.*.id
}

resource "aws_security_group" "sqs_endpoint" {
  count       = var.allow_all_outbound_traffic ? 0 : 1
  name        = "sqs_endpoint_allow_https"
  description = "Allow inbound requests on port 443 for the SQS VPC endpoint."
  vpc_id      = data.aws_vpc.default.id
}

resource "aws_security_group_rule" "sqs_endpoint" {
  count             = var.allow_all_outbound_traffic ? 0 : 1
  type              = "ingress"
  from_port         = 443
  to_port           = 443
  protocol          = "tcp"
  cidr_blocks       = [data.aws_vpc.default.cidr_block]
  security_group_id = aws_security_group.sqs_endpoint.*.id[0]
}

# ---------------------------------------------------------------------------------------------------------------------
# LAUNCH THE OPENVPN HOST
# ---------------------------------------------------------------------------------------------------------------------

# This data source solely exists to simulate a dependency for the module, so that any resources that reference the VPC
# will wait for the VPC endpoint to be created first.
data "template_file" "vpc_id" {
  template = data.aws_vpc.default.id
  depends_on = [
    aws_vpc_endpoint.s3,
    aws_vpc_endpoint.sqs,
  ]
}

module "openvpn" {
  # When using these modules in your own templates, you will need to use a Git URL with a ref attribute that pins you
  # to a specific version of the modules, such as the following example:
  # source = "git::git@github.com:gruntwork-io/module-openvpn.git//modules/openvpn-server?ref=v1.0.0"
  source = "../../modules/openvpn-server"

  aws_account_id = var.aws_account_id
  aws_region     = var.aws_region

  name               = var.name
  instance_type      = "m5.large"
  ami                = var.ami_id
  keypair_name       = var.keypair_name
  user_data          = data.template_file.user_data.rendered
  backup_bucket_name = var.backup_bucket_name

  request_queue_name    = var.request_queue_name
  revocation_queue_name = var.revocation_queue_name
  kms_key_arn           = aws_kms_key.backups.arn
  vpc_id                = data.template_file.vpc_id.rendered
  subnet_id             = data.aws_subnet.default.id

  #WARNING: Only allow SSH from everywhere for test/dev, never in production
  allow_ssh_from_cidr      = true
  allow_ssh_from_cidr_list = ["0.0.0.0/0"]

  #WARNING: Only set this to true for testing/dev, never in production
  backup_bucket_force_destroy = "true"

  allow_all_outbound_traffic = var.allow_all_outbound_traffic
}
