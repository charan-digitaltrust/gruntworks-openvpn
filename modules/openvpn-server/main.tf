# ---------------------------------------------------------------------------------------------------------------------
# CREATE THE ASG
# This defines the number of EC2 Instances to launch
# ---------------------------------------------------------------------------------------------------------------------

resource "aws_autoscaling_group" "openvpn" {
  name = "${var.name}"
  launch_configuration = "${aws_launch_configuration.openvpn.name}"

  desired_capacity = 1
  min_size = 1
  max_size = 1

  vpc_zone_identifier = ["${var.subnet_id}"]

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
  security_groups = ["${aws_security_group.openvpn.id}"]
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

# Create the Security Group for the OpenVPN server
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
  cidr_blocks = ["0.0.0.0/0"]

  security_group_id = "${aws_security_group.openvpn.id}"
}

# Allow SSH access to OpenVPN from the specified Security Group IDs
resource "aws_security_group_rule" "allow_inbound_ssh_security_groups" {
  count = "${var.allow_ssh_from_security_group}"

  type = "ingress"
  from_port = 22
  to_port = 22
  protocol = "tcp"
  source_security_group_id = "${var.allow_ssh_from_security_group_id}"

  security_group_id = "${aws_security_group.openvpn.id}"
}

# Allow SSH access to OpenVPN from the specified CIDR blocks
resource "aws_security_group_rule" "allow_inbound_ssh_cidr_blocks" {
  count = "${var.allow_ssh_from_cidr}"

  type = "ingress"
  from_port = 22
  to_port = 22
  protocol = "tcp"
  cidr_blocks = ["${var.allow_ssh_from_cidr_list}"]

  security_group_id = "${aws_security_group.openvpn.id}"
}

# Allow access to the OpenVPN service from Everywhere
resource "aws_security_group_rule" "allow_inbound_openvpn" {
  type = "ingress"
  from_port = "1194"
  to_port = "1194"
  protocol = "udp"
  cidr_blocks = ["0.0.0.0/0"]

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
    resources = ["*"]
  }

  statement {
    sid = "AssociateAddress"
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

# ----------------------------------------------------------------------------------------------------------------------
# ADD IAM PERMISSIONS TO GIVE THE OPENVPN SERVER ACCESS TO THE REQUEST AND REVOCATION SQS QUEUES
# Here, we give the OpenVPN server permission to assume an IAM role that has access to the SQS queues used for
# certificate requests and revocations. We use an IAM role for this purpose because those SQS queues may be defined in
# a separate AWS account.
# ----------------------------------------------------------------------------------------------------------------------

resource "aws_iam_role_policy" "assume_iam_role_for_queue_access" {
  name   = "assume-iam-role-for-queue-access"
  role   = "${aws_iam_role.openvpn.id}"
  policy = "${data.aws_iam_policy_document.assume_iam_role_for_queue_access.json}"
}

data "aws_iam_policy_document" "assume_iam_role_for_queue_access" {
  statement {
    effect    = "Allow"
    actions   = ["sts:AssumeRole"]
    resources = ["${var.assume_iam_role_arn_for_queue_access}"]
  }
}

