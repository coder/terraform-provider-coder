# terraform-provider-coder

Terraform provider for [Coder](https://github.com/coder/coder).

### Developing

#### Prerequisites

- [Go](https://golang.org/doc/install)
- [Terraform](https://learn.hashicorp.com/tutorials/terraform/install-cli)

We recommend using [`nix`](https://nixos.org/download.html) to manage your development environment. If you have `nix` installed, you can run `nix develop` to enter a shell with all the necessary dependencies.

Alternatively, you can install the dependencies manually.

#### Building

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
4. Run `make build` to build the provider binary, which Terraform will try locate and execute
5. All local Terraform runs will now use your local provider!
6. **NOTE**: We vendor this provider into `github.com/coder/coder`, so if you're testing with a local clone, make sure to run the following in your local clone of `coder`:
   ```console
   go mod edit -replace github.com/coder/terraform-provider-coder/v2=/path/to/terraform-provider-coder
   go mod tidy
   ```
   ⚠️ Be sure to include `/v2` in the module path as it needs to match the version declared in the provider’s `go.mod`.


#### Terraform Acceptance Tests

To run Terraform acceptance tests, run `make testacc`. This will test the provider against the locally installed version of Terraform.

> [!Note]
> Our [CI workflow](./github/workflows/test.yml) runs a test matrix against multiple Terraform versions.

#### Integration Tests

The tests under the `./integration` directory perform the following steps:

- Build the local version of the provider,
- Run an in-memory Coder instance with a specified version,
- Validate the behaviour of the local provider against that specific version of Coder.

To run these integration tests locally:

1. Pull the version of the Coder image you wish to test:

   ```console
     docker pull ghcr.io/coder/coder-preview:main-x.y.z-devel-abcd1234
   ```

1. Run `CODER_IMAGE=ghcr.io/coder/coder-preview CODER_VERSION=main-x.y.z-devel-abcd1234 make test-integration`.

> [!Note]
> You can specify `CODER_IMAGE` if the Coder image you wish to test is hosted somewhere other than `ghcr.io/coder/coder`.
> For example, `CODER_IMAGE=example.com/repo/coder CODER_VERSION=foobar make test-integration`.

### How to create a new release
> [!Warning]
> Before creating a new release, make sure you have pulled the latest commit from the main branch i.e. `git pull origin main`

1. Create a new tag with a version number (following semantic versioning):
   ```console
   git tag -a v2.1.2 -m "v2.1.2"
   ```

2. Push the tag to the remote repository:
   ```console
   git push origin tag v2.1.2
   ```

A GitHub Actions workflow named "Release" will automatically trigger, run integration tests, and publish the new release.
