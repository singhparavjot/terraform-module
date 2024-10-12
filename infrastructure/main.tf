module "sql_server_module" {
  source = "./sql_server_module"
}

module "key_vault_module" {
  source = "./key_vault_module"
}

module "kubernetes_cluster_module" {
  source = "./kubernetes_cluster_module"
}

module "app_gateway_module" {
  source = "./app_gateway_module"
}

