# Getting Started

This provider is available on the [Terraform registry](https://registry.terraform.io/providers/Kong/kong-mesh/latest).

## Sample manifest

Place the following content in `main.tf`, set your `personal_access_token` then run `terraform apply`.

```hcl
# Configure the provider to use your Kong Konnect account
terraform {
  required_providers {
    kong-mesh = {
      source  = "kong/kong-mesh"
    }
  }
}

provider "kong-mesh" {
}
```
