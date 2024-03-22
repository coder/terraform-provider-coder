# terraform-provider-coder

Terraform provider for [Coder](https://github.com/coder/coder).

### Developing

Follow the instructions outlined in the [Terraform documentation](https://developer.hashicorp.com/terraform/cli/config/config-file#development-overrides-for-provider-developers)
to setup your local Terraform to use your local version rather than the registry version.

1. Create a file named `.terraformrc` in your `$HOME` directory
2. Add the following content:
   ```hcl
    provider_installation {
        # Override the coder/coder provider to use your local version
        dev_overrides {
          "coder/coder" = "/path/to/terraform-provider-coder"
        }

        # For all other providers, install them directly from their origin provider
        # registries as normal. If you omit this, Terraform will _only_ use
        # the dev_overrides block, and so no other providers will be available.
        direct {}
    }
   ```
3. (optional, but recommended) Validate your configuration:
    1. Create a new `main.tf` file and include:
      ```hcl
      terraform {
          required_providers {
              coder = {
                  source = "coder/coder"
              }
          }
      }
      ```
   2. Run `terraform init` and observe a warning like `Warning: Provider development overrides are in effect`
4. Run `go build -o terraform-provider-coder` to build the provider binary, which Terraform will try locate and execute
5. All local Terraform runs will now use your local provider!