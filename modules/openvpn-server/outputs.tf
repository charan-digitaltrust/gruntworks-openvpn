output "autoscaling_group_id" {
  value = "${aws_autoscaling_group.openvpn.id}"
}

output "public_ip" {
  value = "${aws_eip.openvpn.public_ip}"
}

output "private_ip" {
  value = "${aws_eip.openvpn.private_ip}"
}

output "elastic_ip" {
  value = "${aws_eip.openvpn.id}"
}

output "security_group_id" {
  value = "${aws_security_group.openvpn.id}"
}

output "iam_role_id" {
  value = "${aws_iam_role.openvpn.id}"
}

output "backup_bucket_name" {
  value = "${lower(var.backup_bucket_name)}"
}

output "assume_iam_role_arn_for_queue_access" {
  value = "${var.assume_iam_role_arn_for_queue_access}"
}