provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "example" {
  name     = "example-resources"
  location = "East US"
}

resource "azurerm_sql_server" "example" {
  name                         = "example-sql-server"
  resource_group_name          = azurerm_resource_group.example.name
  location                     = azurerm_resource_group.example.location
  version                      = "12.0"
  administrator_login          = "adminlogin"
  administrator_login_password = "P@ssw0rd1234"
}

output "sql_server_fully_qualified_domain_name" {
  value = azurerm_sql_server.example.fully_qualified_domain_name
}