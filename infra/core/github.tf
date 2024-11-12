resource "github_actions_variable" "WIF_POOL_NAME" {
  repository    = local.config.github.github_repo
  variable_name = "WIF_POOL_NAME"
  value         = google_iam_workload_identity_pool.wif_pool.workload_identity_pool_id
}

resource "github_actions_variable" "WIF_PROJECT_ID" {
  repository    = local.config.github.github_repo
  variable_name = "WIF_PROJECT_ID"
  value         = local.config.project_id
}

resource "github_actions_variable" "WIF_PROJECT_NUMBER" {
  repository    = local.config.github.github_repo
  variable_name = "WIF_PROJECT_NUMBER"
  value         = local.config.project_number
}

resource "github_actions_variable" "WIF_PROVIDER_NAME" {
  repository    = local.config.github.github_repo
  variable_name = "WIF_PROVIDER_NAME"
  value         = local.config.github.github_provider_id
}

resource "github_actions_variable" "WIF_SA_EMAIL" {
  repository    = local.config.github.github_repo
  variable_name = "WIF_SA_EMAIL"
  value         = google_service_account.wif_service_account.email
}

resource "github_actions_variable" "WIF_SA_PROJECT_ID" {
  repository    = local.config.github.github_repo
  variable_name = "WIF_SA_PROJECT_ID"
  value         = local.config.project_id
}