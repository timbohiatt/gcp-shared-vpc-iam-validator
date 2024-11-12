# INFORMATION 

The IaC (Terraform) in this directory is specifically to be run to only allow for a unified way of managing firewall rules against a shared VPC. 

All firewall rules for each team or subnet are configured in their own folder ```./firewall-rules/{environment}/{service-project}```. Users should raise a PR against the YAML files within their project folder. From there the Automation will take of validating the users have network access on the subnet they wish to modify. 

Additionally it will validate any other expected permissions based on the GitHub Actions Configuration. It will also validate that the Destination IP addresses within the defined rules only relate to the subnet on which they are requesting modification for. Why? Cause firewall rules are not bound to a subnet but rather to a VPC and yet IP ranges must existing within a Subnet and Subnet must be associated with a VPC.

