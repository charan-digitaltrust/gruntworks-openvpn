# ---------------------------------------------------------------------------------------------------------------------
# REQUIRED PARAMETERS
# These variables are expected to be passed in by the operator when calling this terraform module.
# ---------------------------------------------------------------------------------------------------------------------

variable "aws_account_id" {
  description = "The ID of the AWS account in which the resources will be created"
}

variable "aws_region" {
  description = "The AWS region in which the resources will be created"
}

variable "server_iam_role_name" {
  description = "The name to use for the IAM role that will be assumed by the OpenVPN server when it needs to access the request and revocation SQS queues. Note that the IAM role name will be automatically prefixed with 'openvpn-server-'."
}

variable "user_group_name" {
  description = "The name to use for the IAM group for OpenVPN users. Note that the group name will be automatically prefixed with 'openvpn-users-'. Users in this group will be able to request OpenVPN certificates."
}

variable "admin_group_name" {
  description = "The name to use for the IAM group for OpenVPN admins. Note that the group name will be automatically prefixed with 'openvpn-admins-'. Users in this group will be able to revoke OpenVPN certificates."
}

variable "request_queue_name" {
  description = "The name of the sqs queue that will be used to receive new certificate requests. Note that the queue name will be automatically prefixed with 'openvpn-requests-'."
}

variable "revocation_queue_name" {
  description = "The name of the sqs queue that will be used to receive certification revocation requests. Note that the queue name will be automatically prefixed with 'openvpn-revocations-'."
}

# ---------------------------------------------------------------------------------------------------------------------
# OPTIONAL PARAMETERS
# Generally, these values won't need to be changed.
# ---------------------------------------------------------------------------------------------------------------------

variable "external_account_arns_with_openvpn_servers" {
  description = "A list of ARNs external AWS accounts that are running OpenVPN servers and should be given access to the SQS queues in this account. This allows you to define your IAM users in this account and have them use their one set of credentials to request/revoke certificates for OpenVPN servers running in other accounts."
  type        = "list"
  default     = []
}
