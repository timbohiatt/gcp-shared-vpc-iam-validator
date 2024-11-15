resource "random_id" "wif_pool_suffix" {
  keepers = {
    # Generate a new id each time we switch to a new prefix id
    prefix = "${local.config.prefix}"
  }
  byte_length = 8
}

# Create a service account
resource "google_service_account" "wif_service_account" {
  project      = local.config.project_id
  account_id   = "${local.config.prefix}-wif-sa"
  display_name = "${local.config.prefix} WIF Service Account"
}

resource "google_service_account_iam_member" "binding" {
  service_account_id = google_service_account.wif_service_account.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.wif_pool.name}/attribute.repository/${local.config.github.github_user}/${local.config.github.github_repo}"
}

# Create a new Workload Identity Pool
resource "google_iam_workload_identity_pool" "wif_pool" {
  project                   = local.config.project_id
  workload_identity_pool_id = "${local.config.prefix}-${random_id.wif_pool_suffix.hex}-pool"
  provider                  = google-beta
  depends_on                = [google_project_service.project_services]
}

# Create a new Workload Identity Pool Provider
resource "google_iam_workload_identity_pool_provider" "wif_pool_github_provider" {
  project                            = local.config.project_id
  provider                           = google-beta
  workload_identity_pool_id          = google_iam_workload_identity_pool.wif_pool.workload_identity_pool_id
  workload_identity_pool_provider_id = local.config.github.github_provider_id
  oidc {
    issuer_uri = local.config.github.github_issuer_url
  }
  attribute_mapping = {
    "google.subject"       = "assertion.sub"
    "attribute.actor"      = "assertion.actor"
    "attribute.repository" = "assertion.repository"
  }
  attribute_condition = "attribute.repository=='${local.config.github.github_user}/${local.config.github.github_repo}'"
  depends_on = [
    google_iam_workload_identity_pool.wif_pool,
    google_project_service.project_services
  ]
}


# WIF Service account IAM Bindings 
resource "google_project_iam_member" "wif_sa_network_admin" {
  project                            = local.config.project_id
  role    = "roles/compute.networkAdmin"
  member  = "serviceAccount:${google_service_account.wif_service_account.email}"
}