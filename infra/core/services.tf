# Enable Project API's
resource "google_project_service" "project_services" {
  for_each                   = toset(local.config.services)
  project                    = local.config.project_id
  service                    = each.value
  disable_on_destroy         = false
  disable_dependent_services = false
}