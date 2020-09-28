# ---------------------------------------------------------------------------------------------------------------------
# SET TERRAFORM REQUIREMENTS FOR RUNNING THIS MODULE
# This module has been updated with 0.12 syntax, which means it is no longer compatible with any versions below 0.12.
# ---------------------------------------------------------------------------------------------------------------------

terraform {
  # This module is now only being tested with Terraform 0.13.x. However, to make upgrading easier, we are setting
  # 0.12.26 as the minimum version, as that version added support for required_providers with source URLs, making it
  # forwards compatible with 0.13.x code.
  required_version = ">= 0.12.26"
}

# ---------------------------------------------------------------------------------------------------------------------
# CREATE THE ASG
# This defines the number of EC2 Instances to launch
# ---------------------------------------------------------------------------------------------------------------------

resource "aws_autoscaling_group" "openvpn" {
  name                 = var.name
  launch_configuration = aws_launch_configuration.openvpn.name

  desired_capacity = 1
  min_size         = 1
  max_size         = 1

  vpc_zone_identifier = var.subnet_ids

  health_check_type = "EC2"

  tag {
    key                 = "Name"
    value               = var.name
    propagate_at_launch = true
  }

  dynamic "tag" {
    for_each = var.tags
    content {
      key                 = tag.key
      value               = tag.value
      propagate_at_launch = true
    }
  }
}

# ---------------------------------------------------------------------------------------------------------------------
# CREATE THE LAUNCH CONFIGURATION
# This defines the EC2 Instances that will be launched into the Auto Scaling Group
# ---------------------------------------------------------------------------------------------------------------------

# Create the Launch Configuration itself
resource "aws_launch_configuration" "openvpn" {
  # We use the "name_prefix" (versus "name") property to allow a new Launch Configuration to be created without first
  # destroying the old Launch Configuration. This allows a consumer of this module to update the Launch Configuration
  # without destroying and re-creating the Auto Scaling Group.
  name_prefix = "${var.name}-"

  image_id                    = var.ami
  instance_type               = var.instance_type
  key_name                    = var.keypair_name
  user_data                   = var.user_data
  user_data_base64            = var.user_data_base64
  security_groups             = [aws_security_group.openvpn.id]
  iam_instance_profile        = aws_iam_instance_profile.openvpn.name
  associate_public_ip_address = true

  spot_price = var.spot_price

  root_block_device {
    volume_type           = var.root_volume_type
    volume_size           = var.root_volume_size
    iops                  = var.root_volume_iops
    delete_on_termination = var.root_volume_delete_on_termination
  }

  # Important note: whenever using a launch configuration with an auto scaling group, you must set
  # create_before_destroy = true. However, as soon as you set create_before_destroy = true in one resource, you must
  # also set it in every resource that it depends on, or you'll get an error about cyclic dependencies (especially when
  # removing resources). For more info, see:
  #
  # https://www.terraform.io/docs/providers/aws/r/launch_configuration.html
  # https://terraform.io/docs/configuration/resources.html
  lifecycle {
    create_before_destroy = true
  }
}

# Create the Security Group for the OpenVPN server
resource "aws_security_group" "openvpn" {
  name        = var.name
  description = "For OpenVPN instances EC2 Instances."
  vpc_id      = var.vpc_id
  tags        = var.tags

  # See aws_launch_configuration.openvpn for why this directive exists.
  lifecycle {
    create_before_destroy = true
  }
}

# Allow all outbound traffic from the OpenVPN Server
resource "aws_security_group_rule" "allow_outbound_all" {
  type        = "egress"
  from_port   = 0
  to_port     = 0
  protocol    = "-1"
  cidr_blocks = ["0.0.0.0/0"]

  security_group_id = aws_security_group.openvpn.id
}

# Allow SSH access to OpenVPN from the specified Security Group IDs
resource "aws_security_group_rule" "allow_inbound_ssh_security_groups" {
  count = var.allow_ssh_from_security_group ? 1 : 0

  type                     = "ingress"
  from_port                = 22
  to_port                  = 22
  protocol                 = "tcp"
  source_security_group_id = var.allow_ssh_from_security_group_id

  security_group_id = aws_security_group.openvpn.id
}

# Allow SSH access to OpenVPN from the specified CIDR blocks
resource "aws_security_group_rule" "allow_inbound_ssh_cidr_blocks" {
  count = var.allow_ssh_from_cidr ? 1 : 0

  type        = "ingress"
  from_port   = 22
  to_port     = 22
  protocol    = "tcp"
  cidr_blocks = var.allow_ssh_from_cidr_list

  security_group_id = aws_security_group.openvpn.id
}

# Configure inbound VPN access based on var.allow_vpn_from_cidr_list.
resource "aws_security_group_rule" "allow_inbound_openvpn" {
  type        = "ingress"
  from_port   = 1194
  to_port     = 1194
  protocol    = "udp"
  cidr_blocks = var.allow_vpn_from_cidr_list

  security_group_id = aws_security_group.openvpn.id
}

# ---------------------------------------------------------------------------------------------------------------------
# CREATE THE IAM ROLE
# This grants AWS permissions to each EC2 Instance in the cluster.
# ---------------------------------------------------------------------------------------------------------------------

# To assign an IAM Role to an EC2 instance, we actually need to assign the "IAM Instance Profile"
resource "aws_iam_instance_profile" "openvpn" {
  name = var.name
  role = aws_iam_role.openvpn.name

  # See aws_launch_configuration.openvpn for why this directive exists.
  lifecycle {
    create_before_destroy = true
  }

  # There may be a bug where Terraform sometimes doesn't wait long enough for the IAM instance profile to propagate.
  # https://github.com/hashicorp/terraform/issues/4306 suggests it's fixed, but add a "local-exec" provisioner here that
  # sleeps for 30 seconds if this is a problem when running "terraform apply".
  provisioner "local-exec" {
    command = "echo 'Sleeping for 30 seconds to work around IAM Instance Profile propagation bug in Terraform' && python -c 'import time; time.sleep(30)'"
  }
}

# Create the IAM Role where we'll attach permissions
resource "aws_iam_role" "openvpn" {
  name               = var.name
  path               = "/"
  assume_role_policy = data.aws_iam_policy_document.instance_assume_role_policy.json
  tags               = var.tags

  # Workaround for a bug where Terraform sometimes doesn't wait long enough for the IAM role to propagate.
  # https://github.com/hashicorp/terraform/issues/2660
  provisioner "local-exec" {
    command = "echo 'Sleeping for 30 seconds to work around IAM Instance Profile propagation bug in Terraform' && python -c 'import time; time.sleep(30)'"
  }

  # Important note: whenever using a launch configuration with an auto scaling group, you must set
  # create_before_destroy = true. However, as soon as you set create_before_destroy = true in one resource, you must
  # also set it in every resource that it depends on, or you'll get an error about cyclic dependencies (especially when
  # removing resources). For more info, see:
  #
  # https://www.terraform.io/docs/providers/aws/r/launch_configuration.html
  # https://terraform.io/docs/configuration/resources.html
  lifecycle {
    create_before_destroy = true
  }
}

# Use a standard assume-role policy to enable this IAM Role for use with an EC2 Instance
data "aws_iam_policy_document" "instance_assume_role_policy" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type = "Service"

      identifiers = [
        "ec2.amazonaws.com",
      ]
    }
  }
}

# Enable a baseline set of permissions required by OpenVPN
resource "aws_iam_role_policy" "openvpn" {
  name = "${var.name}-allow-default"
  role = aws_iam_role.openvpn.id

  policy = data.aws_iam_policy_document.openvpn.json

  # See aws_launch_configuration.openvpn for why this directive exists.
  lifecycle {
    create_before_destroy = true
  }
}

# Define a baseline set of permissions required by OpenVPN
data "aws_iam_policy_document" "openvpn" {
  statement {
    sid    = "ReadOnlyEC2"
    effect = "Allow"

    actions = [
      "ec2:Describe*",
      "ec2:CreateTags",
      "ec2:DeleteTags",
      "ec2:TerminateInstances",
    ]

    resources = ["*"]
  }

  statement {
    sid    = "AssociateAddress"
    effect = "Allow"

    actions = [
      "ec2:AssociateAddress",
    ]

    resources = ["*"]
  }
}

# ---------------------------------------------------------------------------------------------------------------------
# CREATE AN ELASTIC IP ADDRESS (EIP) FOR THE OPENVPN SERVER
# We output the ID of this EIP so that you can attach the EIP during boot as part of your User Data script
# ---------------------------------------------------------------------------------------------------------------------

resource "aws_eip" "openvpn" {
  count = var.enable_eip ? 1 : 0
  vpc   = true
  tags  = var.tags
}

# ---------------------------------------------------------------------------------------------------------------------
# CREATE THE S3 BACKUP BUCKET
# This bucket is used to store the PKI for OpenVPN for backup purposes should an OpenVPN instance crash
# ---------------------------------------------------------------------------------------------------------------------
resource "aws_s3_bucket" "openvpn" {
  bucket = var.backup_bucket_name

  force_destroy = var.backup_bucket_force_destroy

  versioning {
    enabled = true
  }

  lifecycle_rule {
    enabled = var.enable_backup_bucket_noncurrent_version_expiration

    noncurrent_version_expiration {
      days = var.backup_bucket_noncurrent_version_expiration_days
    }
  }

  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        # If a KMS key is not provided (kms_key_arn is null), the default aws/s3 key is used
        kms_master_key_id = var.kms_key_arn
        sse_algorithm     = "aws:kms"
      }
    }
  }

  tags = merge({
    OpenVPNRole = "BackupBucket"
  }, var.tags)
}

# Block all possibility of accidentally enabling public access to this bucket
resource "aws_s3_bucket_public_access_block" "public_access" {
  bucket                  = aws_s3_bucket.openvpn.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_object" "server-prefix" {
  bucket = aws_s3_bucket.openvpn.bucket
  key    = "server/"
  source = "/dev/null"
}

# ----------------------------------------------------------------------------------------------------------------------
# ADD THE NECESSARY IAM POLICIES TO THE EC2 INSTANCE TO ALLOW BACKUP/RESTORES
# Our cluster EC2 Instance need the ability to read and write to the S3 bucket where backups are stored
# ----------------------------------------------------------------------------------------------------------------------

# Define the IAM Policy Document to be used by the IAM Policy
data "aws_iam_policy_document" "backup" {
  # Important for allowing the OpenVPN instance to read and write objects from S3
  statement {
    sid    = "s3ReadWrite"
    effect = "Allow"

    actions = [
      "s3:Get*",
      "s3:List*",
      "s3:Put*",
    ]

    resources = [
      "arn:aws:s3:::${aws_s3_bucket.openvpn.id}",
      "arn:aws:s3:::${aws_s3_bucket.openvpn.id}/*",
    ]
  }

  statement {
    sid    = "s3ListBuckets"
    effect = "Allow"

    actions = [
      "s3:ListAllMyBuckets",
      "s3:GetBucketTagging",
    ]

    resources = [
      "*",
    ]
  }

  # Encrypt and decrypt objects from S3
  statement {
    sid    = "kmsEncryptDecrypt"
    effect = "Allow"

    actions = [
      "kms:Encrypt",
      "kms:Decrypt",
      "kms:ReEncrypt*",
      "kms:GenerateDataKey*",
      "kms:DescribeKey",
    ]

    resources = [
      var.kms_key_arn,
    ]
  }
}

# Attach the IAM Policy to our IAM Role
resource "aws_iam_role_policy" "backup" {
  name   = "openvpn-backup"
  role   = aws_iam_role.openvpn.id
  policy = data.aws_iam_policy_document.backup.json
}

# ---------------------------------------------------------------------------------------------------------------------
# CREATE THE SQS QUEUES
# This queue is used to receive requests for new certificates
# ---------------------------------------------------------------------------------------------------------------------
resource "aws_sqs_queue" "client-request-queue" {
  name = "openvpn-requests-${var.request_queue_name}"
  tags = var.tags
}

resource "aws_sqs_queue" "client-revocation-queue" {
  name = "openvpn-revocations-${var.revocation_queue_name}"
  tags = var.tags
}

# ----------------------------------------------------------------------------------------------------------------------
# ADD THE NECESSARY IAM POLICIES TO THE EC2 INSTANCE TO ALLOW RECEIVING SQS MESSAGES
# Our cluster EC2 Instance need the ability to recevive messages from the sqs queue to process new client certificate requests
# ----------------------------------------------------------------------------------------------------------------------

# Define the IAM Policy Document to be used by the IAM Policy
data "aws_iam_policy_document" "certificate-requests" {
  # Important for allowing the OpenVPN instance to read and write objects from S3
  statement {
    sid    = "sqsReadDeleteMessages"
    effect = "Allow"

    actions = [
      "sqs:ChangeMessageVisibility",
      "sqs:ChangeMessageVisibilityBatch",
      "sqs:DeleteMessage",
      "sqs:DeleteMessageBatch",
      "sqs:PurgeQueue",
      "sqs:ReceiveMessage",
      "sqs:ReceiveMessageBatch",
    ]

    resources = [
      aws_sqs_queue.client-request-queue.arn,
      aws_sqs_queue.client-revocation-queue.arn,
    ]
  }

  statement {
    sid    = "sqsPublishMessages"
    effect = "Allow"

    actions = [
      "sqs:SendMessage",
      "sqs:SendMessageBatch",
      "sqs:ListQueues",
    ]

    resources = [
      "*",
    ]
  }
}

# Attach the IAM Policy to our IAM Role
resource "aws_iam_role_policy" "certificate-requests" {
  name   = "openvpn-client-requests"
  role   = aws_iam_role.openvpn.id
  policy = data.aws_iam_policy_document.certificate-requests.json
}

# ----------------------------------------------------------------------------------------------------------------------
# CREATE IAM POLICIES THAT ALLOW USERS TO REQUESTS CERTS AND ADMINS TO REVOKE CERTS
# ----------------------------------------------------------------------------------------------------------------------

data "aws_iam_policy_document" "send-certificate-requests" {
  statement {
    sid    = "sqsSendMessages"
    effect = "Allow"

    actions = [
      "sqs:SendMessage",
    ]

    resources = [
      aws_sqs_queue.client-request-queue.arn,
    ]
  }

  statement {
    sid    = "sqsCreateRandomQueue"
    effect = "Allow"

    actions = [
      "sqs:CreateQueue",
      "sqs:DeleteQueue",
      "sqs:ReceiveMessage",
      "sqs:DeleteMessage",
    ]

    resources = [
      "arn:aws:sqs:${var.aws_region}:${var.aws_account_id}:openvpn-response*",
    ]
  }

  statement {
    sid       = "findQueue"
    effect    = "Allow"
    actions   = ["sqs:ListQueues"]
    resources = ["*"]
  }

  statement {
    sid    = "identifyIamUser"
    effect = "Allow"

    actions = [
      "iam:GetUser",
    ]

    resources = [
      # Because AWS IAM Policy Variables (i.e. ${aws:username}) use the same interpolation syntax as Terraform, we have
      # to escape the $ from Terraform with "$$".
      "arn:aws:iam::${var.aws_account_id}:user/$${aws:username}",
    ]
  }
}

resource "aws_iam_policy" "certificate-requests-openvpnusers" {
  name        = "${var.name}-users-certificate-requests"
  description = "Allow OpenVPN users to submit certificate requests via ${aws_sqs_queue.client-request-queue.id}"
  policy      = data.aws_iam_policy_document.send-certificate-requests.json
}

data "aws_iam_policy_document" "send-certificate-revocations" {
  statement {
    sid    = "sqsSendMessages"
    effect = "Allow"

    actions = [
      "sqs:SendMessage",
    ]

    resources = [
      aws_sqs_queue.client-revocation-queue.arn,
    ]
  }

  statement {
    sid       = "findQueue"
    effect    = "Allow"
    actions   = ["sqs:ListQueues"]
    resources = ["*"]
  }
}

resource "aws_iam_policy" "certificate-revocation-openvpnadmins" {
  name        = "${var.name}-admin-certificate-revocations"
  description = "Allow OpenVPN admins to submit certificate revocation requests via ${aws_sqs_queue.client-revocation-queue.id}"
  policy      = data.aws_iam_policy_document.send-certificate-revocations.json
}

# ----------------------------------------------------------------------------------------------------------------------
# ADD IAM GROUPS THAT GIVE USERS ACCESS TO THE SQS QUEUES
# You can add users to these IAM groups to allow them to request or revoke certs.
# ----------------------------------------------------------------------------------------------------------------------

resource "aws_iam_group" "openvpn-users" {
  name = "${var.name}-Users"
}

resource "aws_iam_group" "openvpn-admins" {
  name = "${var.name}-Admins"
}

resource "aws_iam_group_policy_attachment" "certificate-requests" {
  policy_arn = aws_iam_policy.certificate-requests-openvpnusers.arn
  group      = aws_iam_group.openvpn-users.name
}

resource "aws_iam_group_policy_attachment" "revocation-requests" {
  policy_arn = aws_iam_policy.certificate-revocation-openvpnadmins.arn
  group      = aws_iam_group.openvpn-admins.name
}

# ----------------------------------------------------------------------------------------------------------------------
# ADD IAM ROLES THAT GIVE USERS ACCESS TO THE SQS QUEUES
# Users in other AWS accounts can assume these IAM roles to request or revoke certs. Note that these IAM roles are
# only created if the
# ----------------------------------------------------------------------------------------------------------------------

resource "aws_iam_role" "allow_certificate_requests_for_external_accounts" {
  count              = signum(length(var.external_account_arns))
  name               = "${var.name}-allow-certificate-requests-for-external-accounts"
  assume_role_policy = data.aws_iam_policy_document.allow_external_accounts.json
  tags               = var.tags
}

resource "aws_iam_role" "allow_certificate_revocations_for_external_accounts" {
  count              = signum(length(var.external_account_arns))
  name               = "${var.name}-allow-certificate-revocations-for-external-accounts"
  assume_role_policy = data.aws_iam_policy_document.allow_external_accounts.json
  tags               = var.tags
}

data "aws_iam_policy_document" "allow_external_accounts" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "AWS"
      identifiers = var.external_account_arns
    }
  }
}

resource "aws_iam_role_policy_attachment" "allow_certificate_requests_for_external_accounts" {
  count      = signum(length(var.external_account_arns))
  role       = aws_iam_role.allow_certificate_requests_for_external_accounts[0].id
  policy_arn = aws_iam_policy.certificate-requests-openvpnusers.arn
}

resource "aws_iam_role_policy_attachment" "allow_certificate_revocations_for_external_accounts" {
  count      = signum(length(var.external_account_arns))
  role       = aws_iam_role.allow_certificate_revocations_for_external_accounts[0].id
  policy_arn = aws_iam_policy.certificate-revocation-openvpnadmins.arn
}
