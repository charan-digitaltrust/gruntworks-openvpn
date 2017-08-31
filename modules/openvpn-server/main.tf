# ---------------------------------------------------------------------------------------------------------------------
# CREATE THE ASG
# This defines the number of EC2 Instances to launch
# ---------------------------------------------------------------------------------------------------------------------

# Create the Auto Scaling Group
resource "aws_autoscaling_group" "openvpn" {
  name = "${var.name}"
  launch_configuration = "${aws_launch_configuration.openvpn.name}"

  desired_capacity = "1"
  min_size = "1"
  max_size = "1"

  # We only add enough subnets to support the cluster size. This way, we can guarantee that the EBS Volume, ENI, and
  # EC2 Instance all launch in the same Availability Zone, ever for a cluster of size 1. Note that, as documented in the
  # module vars, this requires that users specify their var.vpc_subnet_ids in order of AZ's, starting with the subnet
  # that corresponds to AZ-a, then AZ-b, etc.
  # - The min(..., ...) portion of this code is to handle the case where the Cluster Size is larger than the number
  #   of AZ's. For example, some regions have just 2 AZ's, but we may want a cluster of 3. The slice() function won't
  #   "wrap" the "to index", so we have to ensure the "to index" doesn't exceed the size of the cluster.
  vpc_zone_identifier = [
    "${var.subnet_id}"
  ]

  health_check_type = "EC2"

  tag {
    key = "Name"
    value = "${var.name}"
    propagate_at_launch = true
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

  image_id = "${var.ami}"
  instance_type = "${var.instance_type}"
  key_name = "${var.keypair_name}"
  user_data = "${var.user_data}"
  security_groups = [
    "${aws_security_group.openvpn.id}"]
  iam_instance_profile = "${aws_iam_instance_profile.openvpn.name}"
  associate_public_ip_address = true

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

# Create the Security Group for each MongoDB instance
resource "aws_security_group" "openvpn" {
  name = "${var.name}"
  description = "For OpenVPN instances EC2 Instances."
  vpc_id = "${var.vpc_id}"

  # See aws_launch_configuration.openvpn for why this directive exists.
  lifecycle {
    create_before_destroy = true
  }
}

# Allow all outbound traffic from the OpenVPN Server
resource "aws_security_group_rule" "allow_outbound_all" {
  type = "egress"
  from_port = 0
  to_port = 0
  protocol = "-1"
  cidr_blocks = [
    "0.0.0.0/0"]

  security_group_id = "${aws_security_group.openvpn.id}"
}

# Allow SSH access to OpenVPN from the specified Security Group IDs
resource "aws_security_group_rule" "allow_inbound_ssh_security_groups" {
  count = "${var.allow_ssh_from_security_group}"

  type = "ingress"
  from_port = 22
  to_port = 22
  protocol = "tcp"
  source_security_group_id = "${element(var.allow_ssh_from_security_group_id, count.index)}"

  security_group_id = "${aws_security_group.openvpn.id}"
}

# Allow SSH access to OpenVPN from the specified CIDR blocks
resource "aws_security_group_rule" "allow_inbound_ssh_cidr_blocks" {
  count = "${var.allow_ssh_from_cidr}"

  type = "ingress"
  from_port = 22
  to_port = 22
  protocol = "tcp"
  cidr_blocks = [
    "${var.allow_ssh_from_cidr_list}"]

  security_group_id = "${aws_security_group.openvpn.id}"
}

# Allow access to the OpenVPN service from Everywhere
resource "aws_security_group_rule" "allow_inbound_openvpn" {
  type = "ingress"
  from_port = "1194"
  to_port = "1194"
  protocol = "udp"
  cidr_blocks = [
    "0.0.0.0/0"]

  security_group_id = "${aws_security_group.openvpn.id}"
}

# ---------------------------------------------------------------------------------------------------------------------
# CREATE THE IAM ROLE
# This grants AWS permissions to each EC2 Instance in the cluster.
# ---------------------------------------------------------------------------------------------------------------------

# To assign an IAM Role to an EC2 instance, we actually need to assign the "IAM Instance Profile"
resource "aws_iam_instance_profile" "openvpn" {
  name = "${var.name}"
  role = "${aws_iam_role.openvpn.name}"

  # See aws_launch_configuration.openvpn for why this directive exists.
  lifecycle {
    create_before_destroy = true
  }

  # There may be a bug where Terraform sometimes doesn't wait long enough for the IAM instance profile to propagate.
  # https://github.com/hashicorp/terraform/issues/4306 suggests it's fixed, but add a "local-exec" provisioner here that
  # sleeps for 30 seconds if this is a problem when running "terraform apply".
}

# Create the IAM Role where we'll attach permissions
resource "aws_iam_role" "openvpn" {
  name = "${var.name}"
  path = "/"
  assume_role_policy = "${data.aws_iam_policy_document.instance_assume_role_policy.json}"

  # Workaround for a bug where Terraform sometimes doesn't wait long enough for the IAM role to propagate.
  # https://github.com/hashicorp/terraform/issues/2660
  provisioner "local-exec" {
    command = "echo 'Sleeping for 30 seconds to work around IAM Instance Profile propagation bug in Terraform' && sleep 30"
  }
}

# Use a standard assume-role policy to enable this IAM Role for use with an EC2 Instance
data "aws_iam_policy_document" "instance_assume_role_policy" {
  statement {
    actions = [
      "sts:AssumeRole"]

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
  role = "${aws_iam_role.openvpn.id}"

  policy = "${data.aws_iam_policy_document.openvpn.json}"
}

# Define a baseline set of permissions required by OpenVPN
data "aws_iam_policy_document" "openvpn" {
  statement {
    sid = "ReadOnlyEC2"
    effect = "Allow"
    actions = [
      "ec2:Describe*",
      "ec2:CreateTags",
      "ec2:DeleteTags",
      "ec2:TerminateInstances",
    ]
    resources = [
      "*"]
  }

  statement {
    sid = "AssociateAddress"
    effect = "Allow"
    actions = [
      "ec2:AssociateAddress",
    ]
    resources = [
      "*"
    ]
  }
}

# ------------------------------------------------------------------------------
# CREATE THE USER DATA SCRIPT TO RUN ON THE OPENVPN HOST WHEN IT BOOTS
# ------------------------------------------------------------------------------
data "aws_region" "current" {
  current = true
}

data "aws_vpc" "current" {
  id = "${var.vpc_id}"
}



resource "aws_eip" "openvpn" {
  vpc = true
}

# ---------------------------------------------------------------------------------------------------------------------
# CREATE THE S3 BACKUP BUCKET
# This bucket is used to store the PKI for OpenVPN for backup purposes should an OpenVPN instance crash
# ---------------------------------------------------------------------------------------------------------------------
resource "aws_s3_bucket" "openvpn" {
  bucket = "${var.backup_bucket_name}"

  force_destroy = "${var.backup_bucket_force_destroy}"

  versioning {
    enabled = true
  }
  tags {
    OpenVPNRole = "BackupBucket"
  }
}

resource "aws_s3_bucket_object" "server-prefix" {
  bucket = "${aws_s3_bucket.openvpn.bucket}"
  key = "server/"
  source = "/dev/null"
}

resource "aws_s3_bucket_object" "client-prefix" {
  bucket = "${aws_s3_bucket.openvpn.bucket}"
  key = "client/"
  source = "/dev/null"
}

resource "aws_s3_bucket_object" "request-queue" {
  key = "client/request-queue-url"
  bucket = "${aws_s3_bucket.openvpn.bucket}"
  content = "${aws_sqs_queue.client-request-queue.id}"
  acl = "authenticated-read"
  tags {
    OpenVPNRole = "RequestQueueUrl"
  }
}

resource "aws_s3_bucket_object" "revoke-queue" {
  key = "client/revoke-queue-url"
  bucket = "${aws_s3_bucket.openvpn.bucket}"
  content = "${aws_sqs_queue.client-revocation-queue.id}"
  acl = "authenticated-read"
  tags {
    OpenVPNRole = "RevokeQueueUrl"
  }
}

# ----------------------------------------------------------------------------------------------------------------------
# ADD THE NECESSARY IAM POLICIES TO THE EC2 INSTANCE TO ALLOW BACKUP/RESTORES
# Our cluster EC2 Instance need the ability to read and write to the S3 bucket where backups are stored
# ----------------------------------------------------------------------------------------------------------------------

# Define the IAM Policy Document to be used by the IAM Policy
data "aws_iam_policy_document" "backup" {

  # Important for allowing the OpenVPN instance to read and write objects from S3
  statement {
    sid = "s3ReadWrite"
    effect = "Allow"
    actions = [
      "s3:Get*",
      "s3:List*",
      "s3:Put*"
    ]
    resources = [
      "arn:aws:s3:::${aws_s3_bucket.openvpn.id}",
      "arn:aws:s3:::${aws_s3_bucket.openvpn.id}/*"
    ]
  }

  statement {
    sid = "s3ListBuckets"
    effect = "Allow"
    actions = [
      "s3:ListAllMyBuckets",
      "s3:GetBucketTagging"
    ]
    resources = [
      "*"
    ]
  }

  # Encrypt and decrypt objects from S3
  statement {
    sid = "kmsEncryptDecrypt"
    effect = "Allow"
    actions = [
      "kms:Encrypt",
      "kms:Decrypt",
      "kms:ReEncrypt*",
      "kms:GenerateDataKey*",
      "kms:DescribeKey"
    ]
    resources = [
      "${var.kms_key_arn}"
    ]
  }
}

# Attach the IAM Policy to our IAM Role
resource "aws_iam_role_policy" "backup" {
  name = "openvpn-backup"
  role = "${aws_iam_role.openvpn.id}"
  policy = "${data.aws_iam_policy_document.backup.json}"
}

# ---------------------------------------------------------------------------------------------------------------------
# CREATE THE SQS QUEUES
# This queue is used to receive requests for new certificates
# ---------------------------------------------------------------------------------------------------------------------
resource "aws_sqs_queue" "client-request-queue" {
  name = "${var.request_queue_name}"
}

resource "aws_sqs_queue" "client-revocation-queue" {
  name = "${var.revocation_queue_name}"
}

# ----------------------------------------------------------------------------------------------------------------------
# ADD THE NECESSARY IAM POLICIES TO THE EC2 INSTANCE TO ALLOW RECEIVING SQS MESSAGES
# Our cluster EC2 Instance need the ability to recevive messages from the sqs queue to process new client certificate requests
# ----------------------------------------------------------------------------------------------------------------------

# Define the IAM Policy Document to be used by the IAM Policy
data "aws_iam_policy_document" "certificate-requests" {

  # Important for allowing the OpenVPN instance to read and write objects from S3
  statement {
    sid = "sqsReadDeleteMessages"
    effect = "Allow"
    actions = [
      "sqs:ChangeMessageVisibility",
      "sqs:ChangeMessageVisibilityBatch",
      "sqs:DeleteMessage",
      "sqs:DeleteMessageBatch",
      "sqs:PurgeQueue",
      "sqs:ReceiveMessage",
      "sqs:ReceiveMessageBatch"
    ]
    resources = [
      "${aws_sqs_queue.client-request-queue.arn}",
      "${aws_sqs_queue.client-revocation-queue.arn}"
    ]
  }

  statement {
    sid = "sqsPublishMessages"
    effect = "Allow"
    actions = [
      "sqs:SendMessage",
      "sqs:SendMessageBatch"
    ]
    resources = [
      "*"
    ]
  }
}

# Attach the IAM Policy to our IAM Role
resource "aws_iam_role_policy" "certificate-requests" {
  name = "openvpn-client-requests"
  role = "${aws_iam_role.openvpn.id}"
  policy = "${data.aws_iam_policy_document.certificate-requests.json}"
}

# ----------------------------------------------------------------------------------------------------------------------
# ADD THE NECESSARY IAM GROUPS AND PERMISSIONS FOR OPENVPN USERS AND ADMINS
# ----------------------------------------------------------------------------------------------------------------------
resource "aws_iam_group" "openvpn-users" {
  name = "${var.name}-Users"
}

resource "aws_iam_group" "openvpn-admins" {
  name = "${var.name}-Admins"
}

data "aws_iam_policy_document" "send-certificate-requests" {
  statement {
    sid = "sqsSendMessages"
    effect = "Allow"
    actions = [
      "sqs:SendMessage"
    ]
    resources = [
      "${aws_sqs_queue.client-request-queue.arn}"
    ]
  }

  statement {
    sid = "findOpenVpnBucket"
    effect = "Allow"
    actions = [
      "s3:ListAllMyBuckets",
      "s3:GetBucketTagging"
    ]
    resources = [
      "*"
    ]
  }

  statement {
    sid = "getRequestQueueUrl"
    effect = "Allow"
    actions = [
      "s3:GetObject"
    ]
    resources = [
      "arn:aws:s3:::${aws_s3_bucket.openvpn.id}/client/request-queue-url"
    ]
  }

  statement {
    sid = "identifyIamUser"
    effect = "Allow"
    actions = [
      "iam:GetUser"
    ]
    resources = [
      # Because AWS IAM Policy Variables (i.e. ${aws:username}) use the same interpolation syntax as Terraform, we have
      # to escape the $ from Terraform with "$$".
      "arn:aws:iam::${var.aws_account_id}:user/$${aws:username}"
    ]
  }
}

data "aws_iam_policy_document" "send-certificate-revocations" {
  statement {
    sid = "sqsSendMessages"
    effect = "Allow"
    actions = [
      "sqs:SendMessage"
    ]
    resources = [
      "${aws_sqs_queue.client-revocation-queue.arn}"
    ]
  }

  statement {
    sid = "findOpenVpnBucket"
    effect = "Allow"
    actions = [
      "s3:ListAllMyBuckets",
      "s3:GetBucketTagging"
    ]
    resources = [
      "*"
    ]
  }
}

resource "aws_iam_policy" "certificate-requests-openvpnusers" {
  name = "${var.name}-users-certificate-requests"
  description = "Allow OpenVPN users to submit certificate requests via ${aws_sqs_queue.client-request-queue.id}"
  policy = "${data.aws_iam_policy_document.send-certificate-requests.json}"
}

resource "aws_iam_group_policy_attachment" "certificate-requests" {
  policy_arn = "${aws_iam_policy.certificate-requests-openvpnusers.arn}"
  group = "${aws_iam_group.openvpn-users.name}"
}

resource "aws_iam_policy" "certificate-revocation-openvpnadmins" {
  name = "${var.name}-admin-certificate-revocations"
  description = "Allow OpenVPN admins to submit certificate revocation requests via ${aws_sqs_queue.client-revocation-queue.id}"
  policy = "${data.aws_iam_policy_document.send-certificate-revocations.json}"
}

resource "aws_iam_group_policy_attachment" "revocation-requests" {
  policy_arn = "${aws_iam_policy.certificate-revocation-openvpnadmins.arn}"
  group = "${aws_iam_group.openvpn-admins.name}"
}
