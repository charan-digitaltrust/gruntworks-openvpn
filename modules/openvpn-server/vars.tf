# ---------------------------------------------------------------------------------------------------------------------
# REQUIRED PARAMETERS
# These variables are expected to be passed in by the operator when calling this terraform module.
# ---------------------------------------------------------------------------------------------------------------------

variable "aws_account_id" {
  description = "The AWS account ID where the OpenVPN Server will be created. Note that all IAM Users who receive OpenVPN access must also reside in this AWS account."
  type        = string
}

variable "aws_region" {
  description = "The AWS region in which the resources will be created."
  type        = string
}

variable "name" {
  description = "The name of the server. This will be used to namespace all resources created by this module."
  type        = string
}

variable "backup_bucket_name" {
  description = "The name of the s3 bucket that will be used to backup PKI secrets"
  type        = string
}

variable "request_queue_name" {
  description = "The name of the sqs queue that will be used to receive new certificate requests. Note that the queue name will be automatically prefixed with 'openvpn-requests-'."
  type        = string
}

variable "revocation_queue_name" {
  description = "The name of the sqs queue that will be used to receive certification revocation requests. Note that the queue name will be automatically prefixed with 'openvpn-revocations-'."
  type        = string
}

variable "kms_key_arn" {
  description = "The Amazon Resource Name (ARN) of the KMS Key that will be used to encrypt/decrypt backup files."
  type        = string
}

variable "ami" {
  description = "The ID of the AMI to run for this server."
  type        = string
}

variable "vpc_id" {
  description = "The id of the VPC where this server should be deployed."
  type        = string
}

variable "subnet_id" {
  description = "The id of the subnet where this server should be deployed."
  type        = string
}

variable "keypair_name" {
  description = "The name of a Key Pair that can be used to SSH to this instance. Leave blank if you don't want to enable Key Pair auth."
  type        = string
}

variable "instance_type" {
  description = "The type of EC2 instance to run (e.g. t2.micro)"
  type        = string
}

variable "user_data" {
  description = "The User Data script to run on this instance when it is booting."
  type        = string
}

# ---------------------------------------------------------------------------------------------------------------------
# OPTIONAL PARAMETERS
# Generally, these values won't need to be changed.
# ---------------------------------------------------------------------------------------------------------------------
#
variable "allow_vpn_from_cidr_list" {
  description = "A list of IP address ranges in CIDR format from which VPN access will be permitted. Attempts to access the VPN server from all other IP addresses will be blocked."
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

variable "allow_ssh_from_cidr" {
  description = "A boolean that specifies if this server will allow SSH connections from the list of CIDR blocks specified in var.allow_ssh_from_cidr_list."
  type        = bool
  default     = false
}

variable "allow_ssh_from_cidr_list" {
  description = "A list of IP address ranges in CIDR format from which SSH access will be permitted. Attempts to access the VPN server from all other IP addresses will be blocked. This is only used if var.allow_ssh_from_cidr is true."
  type        = list(string)
  default     = []
}

variable "allow_ssh_from_security_group" {
  description = "A boolean that specifies if this server will allow SSH connections from the security group specified in var.allow_ssh_from_security_group_id."
  type        = bool
  default     = false
}

variable "allow_ssh_from_security_group_id" {
  description = "The ID of a security group from which SSH connections will be allowed. Only used if var.allow_ssh_from_security_group is true."
  type        = string
  default     = null
}

variable "backup_bucket_force_destroy" {
  description = "When a terraform destroy is run, should the backup s3 bucket be destroyed even if it contains files. Should only be set to true for testing/development"
  type        = bool
  default     = false
}

variable "enable_backup_bucket_noncurrent_version_expiration" {
  description = "Should lifecycle policy to expire noncurrent versions be enabled."
  type        = bool
  default     = false
}

variable "backup_bucket_noncurrent_version_expiration_days" {
  description = "Number of days that non current versions of file should be kept. Only used if var.enable_backup_bucket_noncurrent_version_expiration is true"
  type        = number
  default     = 30
}

variable "tenancy" {
  description = "The tenancy of this server. Must be one of: default, dedicated, or host."
  type        = string
  default     = "default"
}

variable "external_account_arns" {
  description = "The ARNs of external AWS accounts where your IAM users are defined. If not empty, this module will create IAM roles that users in those accounts will be able to assume to get access to the request/revocation SQS queues."
  type        = list(string)
  default     = []
}

variable "root_volume_type" {
  description = "The root volume type. Must be one of: standard, gp2, io1."
  type        = string
  default     = "gp2"
}

variable "root_volume_size" {
  description = "The size of the root volume, in gigabytes."
  type        = number
  default     = 8
}

variable "root_volume_iops" {
  description = "The amount of provisioned IOPS. This is only valid for volume_type of io1, and must be specified if using that type."
  type        = number
  default     = 0
}

variable "root_volume_delete_on_termination" {
  description = "If set to true, the root volume will be deleted when the Instance is terminated."
  type        = bool
  default     = true
}

variable "enable_eip" {
  description = "When set to true AWS will create an eip for the OpenVPN server and output it so it can be attached during boot with the user data script when set to false no eip will be created"
  type        = bool
  default     = true
}

variable "spot_price" {
  description = "Set this parameter to use spot instances for your OpenVPN server. This parameter controls the maximum price to use for reserving spot instances. This can save you a lot of money on the VPN server, but it also risks that the server will be down if your requested spot instance price cannot be met."
  type        = number
  default     = null
}
