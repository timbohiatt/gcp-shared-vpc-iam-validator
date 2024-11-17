# Configure the Terraform backend
terraform {
  backend "gcs" {
    bucket = "tf-state-493242"
    prefix = "terraform/state"
  }

}

locals {
  yaml_dir = "../../firewall-rules"
}

module "vpc-trusted-internal-firewall" {
  source     = "../modules/net-vpc-firewall"
  project_id = "thiatt-manual-124"
  network    = "shared-vpc"
  default_rules_config = {
    disabled = true
  }
  factories_config = {
    //cidr_tpl_file = "${var.factories_config.data_dir}/networking/cidrs.yaml"
    rules_folder = "${local.yaml_dir}"
  }
}
