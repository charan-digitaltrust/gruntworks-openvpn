# OpenVPN User Management Module

This module allows you to manage who has access to your [OpenVPN Server](/modules/openvpn-server) using two IAM groups:

1. **Users**: This group is for normal OpenVPN users. Any user added to this group will be able to request a 
   certificate using [openvpn-admin](/modules/openvpn-admin) and use that certificate to connect to the OpenVPN server.
   
1. **Admins**: This group is for OpenVPN administrators. Any user in this group will be able to revoke certificates
   [openvpn-admin](/modules/openvpn-admin) so a user no longer has access to the OpenVPN server.
   
Under the hood, this module creates two SQS queues used by `openvpn-admin`: one used to process certificate requests 
and one used to process certificate revocations. The `openvpn-server` listens on messages on both of these queues and
processes them accordingly.         




## How do you use this module?

* See the [root README](/README.md) for instructions on using Terraform modules.
* See the [examples](/examples) folder for example usage.
* See [vars.tf](./vars.tf) for all the variables you can set on this module.




## How do you use this module with multiple AWS accounts?

If you have multiple AWS accounts and your IAM users are defined in a different account than the OpenVPN server, you'll
need to deploy things as described below. The code snippets below assume that your IAM users are defined in an AWS 
account with ID 11111111111111 and that you are deploying two OpenVPN servers, one in an AWS account with ID 
22222222222222 and one in an AWS account with ID 33333333333333.

1. Deploy the `openvpn-user-mgmt` module in the account where your IAM users are defined, making sure to use the
   `external_account_arns_with_openvpn_servers` variable to specify the ARNs of the other AWS account(s) where you plan 
   to run OpenVPN servers:
   
    ```hcl
    module "openvpn_user_mgmt" {
      source = "git::git@github.com:gruntwork-io/module-openvpn.git//modules/openvpn-user-mgmt?ref=v1.0.0"
   
      account_id = "11111111111111"
   
      # These are the ARNs of other AWS accounts that will be running an OpenVPN server
      external_account_arns_with_openvpn_servers = [
        "arn:aws:iam::22222222222222:root",
        "arn:aws:iam::33333333333333:root"
      ]
   
      # ... (other params omitted) ...
    }
    ```
    
    This will create the two IAM groups (one for OpenVPN users, one for admins) in the account where your IAM users are 
    defined so you can add those users to the groups to give them OpenVPN access. This module will output the ARN of
    an IAM role using the `openvpn_server_iam_role_arn` variable.
    
1. To run an OpenVPN server in the other AWS accounts, deploy the `openvpn-server` module in each one, making sure to 
   set the `assume_iam_role_arn_for_queue_access` variable to the `openvpn_server_iam_role_arn` output from the 
   previous step:
   
    ```hcl
    module "openvpn" {
      source = "git::git@github.com:gruntwork-io/module-openvpn.git//modules/openvpn-server?ref=v1.0.0"
   
      account_id = "22222222222222"
   
      assume_iam_role_arn_for_queue_access = "arn:aws:iam::11111111111111:role/openvpn-server-security-account"
   
      # ... (other params omitted) ...
    }
    ```
    
    Now the OpenVPN server in account 22222222222222 will be able to read and write from the SQS queues in account
    11111111111111, which is where your IAM users (and those queues) are defined.