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
  default     = "us-east-1"
}

variable "aws_account_id" {
  description = "The AWS account ID where the OpenVPN Server will be created. Note that all IAM Users who receive OpenVPN access must also reside in this AWS account."
}

variable "keypair_name" {
  description = "The AWS EC2 Keypair name for root access to the OpenVPN host."
  default     = ""
}

variable "backup_bucket_name" {
  description = "The name of the s3 bucket that will hold the backup of the PKI for the OpenVPN server"
  default     = "openvpn-backups"
}

variable "request_queue_name" {
  description = "The name of the sqs queue that will be used to receive new certificate requests. Note that the queue name will be automatically prefixed with 'openvpn-requests-'."
  default     = "example"
}

variable "revocation_queue_name" {
  description = "The name of the sqs queue that will be used to receive certificate revocation requests. Note that the queue name will be automatically prefixed with 'openvpn-revocations-'."
  default     = "example"
}

variable "name" {
  description = "The name of the openvpn host"
  default     = "openvpn-host"
}

variable "ami_id" {
  description = "The ID of the AMI to run. Should be an AMI built from the Packer template in /examples/packer/build.json"
}
