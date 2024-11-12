locals {
  yaml_dir = "../../firewall-rules"
}

# data "utils_deep_merge_yaml" "merged_yaml" {
#   input = [for file in fileset(local.yaml_dir, "**/*.yaml") : file("${local.yaml_dir}/${file}")]
# }

# output "applied_firewall_rules" {
#   value = data.utils_deep_merge_yaml.merged_yaml.output
# }

module "vpc-trusted-internal-firewall" {
  source     = "../modules/net-vpc-firewall"
  project_id = "thiatt-manual-120"
  network    = "sample-vpc-global"
  default_rules_config = {
    disabled = true
  }
  factories_config = {
    //cidr_tpl_file = "${var.factories_config.data_dir}/networking/cidrs.yaml"
    rules_folder  = "${local.yaml_dir}"
  }
}
