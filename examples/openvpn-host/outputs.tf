output "openvpn_host_public_ip" {
  value = "${module.openvpn.public_ip}"
}

output "openvpn_host_security_group_id" {
  value = "${module.openvpn.security_group_id}"
}
