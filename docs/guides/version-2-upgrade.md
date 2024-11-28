---
page_title: "Version 2 Upgrade Guide"
---

# Version 2 Upgrade Guide

Version 2.0.0 of the Coder provider for Terraform is a major release that introduces some changes that you will need to consider when upgrading.
This guide is intended to help with the process, and focuses only on the changes from version 1.X to version 2.0.0.

!> Using Version 2.0.0 of the Coder provider requires Coder Server version [`2.18.0`](https://github.com/coder/coder/releases/tag/v2.18.0) or later.

Upgrade topics:

- [Provider Version Configuration](#provider-version-configuration)
- [Provider Arguments](#provider-arguments)
- [Data Source: [`coder_git_auth`](#data-source-coder_git_auth)
- [Data Source: [`coder_workspace`](#data-source-coder_workspace)

## Provider Version Configuration

-> Before upgrading to version 2.0.0, please first upgrade to the most recent 1.X version and ensure that your environment successfully runs [`terraform plan`](https://developer.hashicorp.com/terraform/cli/commands/plan) without unexpected changes or deprecation notices.

We highly recommend using [version constraints](https://developer.hashicorp.com/terraform/language/providers/requirements#version-constraints) when configuring Terraform providers.


For example, given the previous configuration:

```terraform
terraform {
  required_providers {
    coder = {
      source  = "coder/coder"
      version = "~> 1.0.0"
    }
  }
}

provider "coder" {
  feature_use_managed_variables = true
}
```

Update to the latest 2.X version:

```terraform
terraform {
  required_providers {
    coder = {
      source  = "coder/coder"
      version = "~> 2.0.0"
    }
  }
}

provider "coder" {}
```

## Provider Arguments

Version 2.0.0 removes the [`feature_use_managed_variables`](https://registry.terraform.io/providers/coder/coder/1.0.4/docs#feature_use_managed_variables-1) argument from the `provider` block.


## Data Source: [`coder_git_auth`](https://registry.terraform.io/providers/coder/coder/1.0.4/docs/data-sources/git_auth)

If you are using this data source, you must replace it with the [`coder_external_auth`](https://registry.terraform.io/providers/coder/coder/2.0.0/docs/data-sources/external_auth) data source. The `coder_external_auth` data source is a more generic data source that can be used to create any external authentication provider which supports OAuth2.

For example, given the previous configuration:

```terraform
data "coder_git_auth" "example" {
  id = "example"
}
```

Update to the new data source:

```terraform
data "coder_external_auth" "example" {
  id = "example"
}
```

## Data Source: [`coder_workspace`](https://registry.terraform.io/providers/coder/coder/1.0.4/docs/data-sources/workspace)

If you are using the `owner` properties of the `coder_workspace` data source, you must remove them and use the [`coder_workspace_owner`](https://registry.terraform.io/providers/coder/coder/2.0.0/docs/data-sources/workspace_owner) data source instead. The `coder_workspace_owner` data source provides additional properties of the workspace owner.

Update your Terraform configuration to use the `coder_workspace_owner` data source instead and update the following attributes:

```terraform
data "coder_workspace_owner" "me" {}
```

- Remove `owner_id` attribute. Use [`data.coder_workspace_owner.me.id`](https://registry.terraform.io/providers/coder/coder/2.0.0/docs/data-sources/workspace_owner#id) instead.
- Remove `owner` attribute. Use [`data.coder_workspace_owner.me.name`](https://registry.terraform.io/providers/coder/coder/2.0.0/docs/data-sources/workspace_owner#name) instead.
- Remove `owner_name` attribute. Use [`data.coder_workspace_owner.me.full_name`](https://registry.terraform.io/providers/coder/coder/2.0.0/docs/data-sources/workspace_owner#full_name) instead.
- Remove `owner_email` attribute. Use [`data.coder_workspace_owner.me.email`](https://registry.terraform.io/providers/coder/coder/2.0.0/docs/data-sources/workspace_owner#email) instead.
- Remove `owner_groups` attribute. Use [`data.coder_workspace_owner.me.groups`](https://registry.terraform.io/providers/coder/coder/2.0.0/docs/data-sources/workspace_owner#groups) instead.
- Remove `owner_oidc_access_token` attribute. Use [`data.coder_workspace_owner.me.oidc_access_token`](https://registry.terraform.io/providers/coder/coder/2.0.0/docs/data-sources/workspace_owner#oidc_access_token) instead.
- Remove `owner_session_token` attribute. Use [`data.coder_workspace_owner.me.session_token`](https://registry.terraform.io/providers/coder/coder/2.0.0/docs/data-sources/workspace_owner#session_token) instead.

->While we do not anticipate these changes to affect existing resources, we strongly advice reviewing the plan produced by Terraform to ensure no resources are accidentally removed or altered in an undesired way. If you encounter any unexpected behavior, please report it by opening a GitHub [issue](https://github.com/coder/terraform-provider-coder/issues).