output "client_request_queue" {
  value = "${aws_sqs_queue.client_request_queue.id}"
}

output "client_revocation_queue" {
  value = "${aws_sqs_queue.client_revocation_queue.id}"
}

output "user_group_name" {
  value = "${aws_iam_group.openvpn_users.name}"
}

output "admin_group_name" {
  value = "${aws_iam_group.openvpn_admins.name}"
}

output "openvpn_server_iam_role_id" {
  value = "${aws_iam_role.openvpn_server.id}"
}

output "openvpn_server_iam_role_arn" {
  value = "${aws_iam_role.openvpn_server.arn}"
}