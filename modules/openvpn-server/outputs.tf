output "autoscaling_group_id" {
  value = "${aws_autoscaling_group.openvpn.id}"
}

output "public_ip" {
  value = "${aws_eip.openvpn.public_ip}"
}

output "private_ip" {
  value = "${aws_eip.openvpn.private_ip}"
}

output "security_group_id" {
  value = "${aws_security_group.openvpn.id}"
}

output "iam_role_id" {
  value = "${aws_iam_role.openvpn.id}"
}

output "client_request_queue" {
  value = "${aws_sqs_queue.client-request-queue.id}"
}