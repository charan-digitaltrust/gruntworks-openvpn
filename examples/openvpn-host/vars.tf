# ---------------------------------------------------------------------------------------------------------------------
# ENVIRONMENT VARIABLES
# Define these secrets as environment variables
# ---------------------------------------------------------------------------------------------------------------------

# AWS_ACCESS_KEY_ID
# AWS_SECRET_ACCESS_KEY

# ---------------------------------------------------------------------------------------------------------------------
# MODULE PARAMETERS
# These variables are expected to be passed in by the operator
# ---------------------------------------------------------------------------------------------------------------------

variable "aws_region" {
  description = "The AWS region in which all resources will be created"
  default = "us-east-1"
}

variable "keypair_name" {
  description = "The AWS EC2 Keypair name for root access to the OpenVPN host."
  default = ""
}

variable "backup_bucket_name" {
  description = "The name of the s3 bucket that will hold the backup of the PKI for the OpenVPN server"
  default = "openvpn-backups"
}

variable "request_queue_name" {
  description = "The name of the sqs queue that will be used to receive new certificate requests"
  default = "openvpn-requests"
}

variable "revocation_queue_name" {
  description = "The name of the sqs queue that will be used to receive certificate revocation requests"
  default = "openvpn-revokes"
}

variable "name" {
  description = "The name of the openvpn host"
  default = "openvpn-host"
}

variable "ami" {
  description = "The ID of the AMI to run. Should be built with packer scripts in /examples/packer"
}

variable "backup_kms_key" {
  description = "The KMS key to use for asset encryption/decryption"
}
