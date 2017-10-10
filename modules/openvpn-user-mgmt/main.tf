# ---------------------------------------------------------------------------------------------------------------------
# CREATE THE REQUEST AND REVOKE SQS QUEUES
# These queues are used to receive requests for new certificates and for revoking existing certificates
# ---------------------------------------------------------------------------------------------------------------------

resource "aws_sqs_queue" "client_request_queue" {
  name = "openvpn-${var.request_queue_name}"
}

resource "aws_sqs_queue" "client_revocation_queue" {
  name = "openvpn-${var.revocation_queue_name}"
}

# ----------------------------------------------------------------------------------------------------------------------
# CREATE AN IAM ROLE THAT ALLOWS THE OPENVPN SERVER TO USE THE REQUEST AND REVOKE SQS QUEUES
# The OpenVPN Server will need to assume this IAM role to process certificate request/revoke messages. Note that we've
# configured this as an IAM Role so that we can handle permissions the same way whether the OpenVPN server is defined
# in this AWS account or another one.
# ----------------------------------------------------------------------------------------------------------------------

resource "aws_iam_role" "openvpn_server" {
  assume_role_policy = "${length(var.external_account_arns_with_openvpn_servers) > 0 ? data.aws_iam_policy_document.openvpn_server_multiple_accounts.json : data.aws_iam_policy_document.openvpn_server_this_account_only.json}"
}

data "aws_iam_policy_document" "openvpn_server_this_account_only" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ec2.amazonaws.com"]
    }
  }
}

data "aws_iam_policy_document" "openvpn_server_multiple_accounts" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ec2.amazonaws.com"]
    }
  }

  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "AWS"
      identifiers = ["${var.external_account_arns_with_openvpn_servers}"]
    }
  }
}

resource "aws_iam_role_policy" "openvpn_server_use_request_revoke_queues" {
  name   = "use-request-revoke-queues"
  role   = "${aws_iam_role.openvpn_server.id}"
  policy = "${data.aws_iam_policy_document.openvpn_server_use_request_revoke_queues}"
}

data "aws_iam_policy_document" "openvpn_server_use_request_revoke_queues" {
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
      "${aws_sqs_queue.client_request_queue.arn}",
      "${aws_sqs_queue.client_revocation_queue.arn}"
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

# ----------------------------------------------------------------------------------------------------------------------
# CREATE AN IAM GROUP AND PERMISSIONS FOR OPENVPN USERS
# Users in this IAM group will be able to request OpenVPN certificates
# ----------------------------------------------------------------------------------------------------------------------

resource "aws_iam_group" "openvpn_users" {
  name = "openvpn-${var.user_group_name}"
}

resource "aws_iam_group_policy" "send_certificate_requests" {
  name   = "send-certificate-requests"
  group  = "${aws_iam_group.openvpn_users.name}"
  policy = "${data.aws_iam_policy_document.send_certificate_requests.json}"
}

data "aws_iam_policy_document" "send_certificate_requests" {
  statement {
    sid = "sqsSendMessages"
    effect = "Allow"
    actions = [
      "sqs:SendMessage"
    ]
    resources = [
      "${aws_sqs_queue.client_request_queue.arn}"
    ]
  }

  statement {
    sid = "sqsCreateRandomQueue"
    effect = "Allow"
    actions = [
      "sqs:CreateQueue",
      "sqs:DeleteQueue",
      "sqs:ReceiveMessage",
      "sqs:DeleteMessage"
    ]
    resources = [
      "arn:aws:sqs:${var.aws_region}:${var.aws_account_id}:openvpn-response*"
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

# ----------------------------------------------------------------------------------------------------------------------
# CREATE AN IAM GROUP AND PERMISSIONS FOR OPENVPN ADMINS
# Users in this IAM group will be able to revoke OpenVPN certificates
# ----------------------------------------------------------------------------------------------------------------------

resource "aws_iam_group" "openvpn_admins" {
  name = "openvpn-${var.admin_group_name}"
}

resource "aws_iam_group_policy" "send_certificate_revocations" {
  name   = "send-certificate-revocations"
  group  = "${aws_iam_group.openvpn_admins.name}"
  policy = "${data.aws_iam_policy_document.send_certificate_revocations.json}"
}

data "aws_iam_policy_document" "send_certificate_revocations" {
  statement {
    sid = "sqsSendMessages"
    effect = "Allow"
    actions = [
      "sqs:SendMessage"
    ]
    resources = [
      "${aws_sqs_queue.client_revocation_queue.arn}"
    ]
  }
}
