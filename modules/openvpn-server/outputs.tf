output "autoscaling_group_id" {
  value = aws_autoscaling_group.openvpn.id
}

output "public_ip" {
  value = element(concat(aws_eip.openvpn.*.public_ip, [""]), 0)
}

output "private_ip" {
  value = element(concat(aws_eip.openvpn.*.private_ip, [""]), 0)
}

output "elastic_ip" {
  value = element(concat(aws_eip.openvpn.*.id, [""]), 0)
}

output "security_group_id" {
  value = aws_security_group.openvpn.id
}

output "iam_role_id" {
  value = aws_iam_role.openvpn.id
}

output "client_request_queue" {
  value = aws_sqs_queue.client-request-queue.id
}

output "client_revocation_queue" {
  value = aws_sqs_queue.client-revocation-queue.id
}

output "backup_bucket_name" {
  value = lower(var.backup_bucket_name)
}

output "allow_certificate_requests_for_external_accounts_iam_role_id" {
  value = element(concat(aws_iam_role.allow_certificate_requests_for_external_accounts.*.id, [""]), 0)
}

output "allow_certificate_requests_for_external_accounts_iam_role_arn" {
  value = element(concat(aws_iam_role.allow_certificate_requests_for_external_accounts.*.arn, [""]), 0)
}

output "allow_certificate_revocations_for_external_accounts_iam_role_id" {
  value = element(concat(aws_iam_role.allow_certificate_revocations_for_external_accounts.*.id, [""]), 0)
}

output "allow_certificate_revocations_for_external_accounts_iam_role_arn" {
  value = element(concat(aws_iam_role.allow_certificate_revocations_for_external_accounts.*.arn, [""]), 0)
}

output "openvpn_users_group_name" {
  value = aws_iam_group.openvpn-users.name
}

output "openvpn_admins_group_name" {
  value = aws_iam_group.openvpn-admins.name
}
