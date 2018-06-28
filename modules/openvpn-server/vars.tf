# ---------------------------------------------------------------------------------------------------------------------
# REQUIRED PARAMETERS
# These variables are expected to be passed in by the operator when calling this terraform module.
# ---------------------------------------------------------------------------------------------------------------------

variable "aws_account_id" {
  description = "The AWS account ID where the OpenVPN Server will be created. Note that all IAM Users who receive OpenVPN access must also reside in this AWS account."
}

variable "aws_region" {
  description = "The AWS region in which the resources will be created."
}

variable "name" {
  description = "The name of the server. This will be used to namespace all resources created by this module."
}

variable "backup_bucket_name" {
  description = "The name of the s3 bucket that will be used to backup PKI secrets"
}

variable "request_queue_name" {
  description = "The name of the sqs queue that will be used to receive new certificate requests. Note that the queue name will be automatically prefixed with 'openvpn-requests-'."
}

variable "revocation_queue_name" {
  description = "The name of the sqs queue that will be used to receive certification revocation requests. Note that the queue name will be automatically prefixed with 'openvpn-revocations-'."
}

variable "kms_key_arn" {
  description = "The Amazon Resource Name (ARN) of the KMS Key that will be used to encrypt/decrypt backup files."
}

variable "ami" {
  description = "The ID of the AMI to run for this server."
}

variable "vpc_id" {
  description = "The id of the VPC where this server should be deployed."
}

variable "subnet_id" {
  description = "The id of the subnet where this server should be deployed."
}

variable "keypair_name" {
  description = "The name of a Key Pair that can be used to SSH to this instance. Leave blank if you don't want to enable Key Pair auth."
}

variable "instance_type" {
  description = "The type of EC2 instance to run (e.g. t2.micro)"
}

variable "user_data" {
  description = "The User Data script to run on this instance when it is booting."
}

# ---------------------------------------------------------------------------------------------------------------------
# OPTIONAL PARAMETERS
# Generally, these values won't need to be changed.
# ---------------------------------------------------------------------------------------------------------------------

variable "allow_ssh_from_cidr" {
  description = "A boolean that specifies if this server will allow SSH connections from the list of CIDR blocks specified in var.allow_ssh_from_cidr_list."
  default     = false
}

variable "allow_ssh_from_cidr_list" {
  description = "A list of IP address ranges in CIDR format from which SSH access will be permitted. Attempts to access the bastion host from all other IP addresses will be blocked. This is only used if var.allow_ssh_from_cidr is true."
  type        = "list"
  default     = []
}

variable "allow_ssh_from_security_group" {
  description = "A boolean that specifies if this server will allow SSH connections from the security group specified in var.allow_ssh_from_security_group_id."
  default     = false
}

variable "allow_ssh_from_security_group_id" {
  description = "The ID of a security group from which SSH connections will be allowed. Only used if var.allow_ssh_from_security_group is true."
  default     = ""
}

variable "backup_bucket_force_destroy" {
  description = "When a terraform destroy is run, should the backup s3 bucket be destroyed even if it contains files. Should only be set to true for testing/development"
  default     = false
}

variable "tenancy" {
  description = "The tenancy of this server. Must be one of: default, dedicated, or host."
  default     = "default"
}

variable "external_account_arns" {
  description = "The ARNs of external AWS accounts where your IAM users are defined. If not empty, this module will create IAM roles that users in those accounts will be able to assume to get access to the request/revocation SQS queues."
  type        = "list"
  default     = []
}

variable "root_volume_type" {
  description = "The root volume type. Must be one of: standard, gp2, io1."
  default = "standard"
}

variable "root_volume_size" {
  description = "The size of the root volume, in gigabytes."
  default = 8
}

variable "root_volume_iops" {
  description = "The amount of provisioned IOPS. This is only valid for volume_type of io1, and must be specified if using that type."
  default = 0
}

variable "root_volume_delete_on_termination" {
  description = "If set to true, the root volume will be deleted when the Instance is terminated."
  default = true
}
