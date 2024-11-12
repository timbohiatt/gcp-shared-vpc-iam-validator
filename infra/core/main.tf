//gratitude-state-dev-e1fccc5c

# Configure the Terraform backend
terraform {
  backend "gcs" {
    // bucket = XXXXXX  <---- This is set at the Command Line
    bucket = "vpc-fw-automation-state-dev-e1fcsc5c"
    prefix = "terraform/state"
  }

}

locals {
  config = {
    env            = "dev"
    prefix         = "svfw-auto"
    project_id     = "thiatt-manual-121"
    project_number = "633533567477"
    services = [
      "compute.googleapis.com",
      "networkservices.googleapis.com",
      "iam.googleapis.com",
    ]
    github = {
      github_user        = "timbohiatt"
      github_repo        = "gcp-shared-vpc-iam-validator"
      github_provider_id = "github"
      github_issuer_url  = "https://token.actions.githubusercontent.com"
    }
  }
}