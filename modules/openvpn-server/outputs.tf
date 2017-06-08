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

output "client_request_queue" {
  value = "${aws_sqs_queue.client-request-queue.id}"
}

output "client_revocation_queue" {
  value = "${aws_sqs_queue.client-revocation-queue.id}"
}

output "backup_bucket_name" {
  value = "${lower(var.backup_bucket_name)}"
}

output "key_size" {
  value = "${var.openssl_key_size}"
}

output "openssl_ca_expiration_days" {
  value = "${var.openssl_ca_expiration_days}"
}

output "openssl_certificate_expiration_days" {
  value = "${var.openssl_certificate_expiration_days}"
}

output "ca_country" {
  value = "${var.ca_country}"
}

output "ca_state" {
  value = "${var.ca_state}"
}

output "ca_locality" {
  value = "${var.ca_locality}"
}

output "ca_org" {
  value = "${var.ca_org}"
}

output "ca_org_unit" {
  value = "${var.ca_org_unit}"
}

output "ca_email" {
  value = "${var.ca_email}"
}

output "vpn_routes" {
  value = "${var.vpn_routes}"
}
