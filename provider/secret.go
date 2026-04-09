package provider

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/coder/terraform-provider-coder/v2/provider/helpers"
)

// secretDataSource returns a schema for a user secret data source.
func secretDataSource() *schema.Resource {
	return &schema.Resource{
		SchemaVersion: 1,

		Description: "Use this data source to declare that a workspace requires a user secret. " +
			"Each `coder_secret` block declares a single secret requirement, matched by either " +
			"an environment variable name (`env`) or a file path (`file`). The resolved value " +
			"is available at build time via `data.coder_secret.<name>.value`.",
		ReadContext: func(ctx context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
			env := rd.Get("env").(string)
			file := rd.Get("file").(string)

			if env == "" && file == "" {
				return diag.Errorf("exactly one of `env` or `file` must be set")
			}
			if env != "" && file != "" {
				return diag.Errorf("exactly one of `env` or `file` must be set")
			}

			// Build a stable ID from whichever field is set.
			if env != "" {
				rd.SetId(fmt.Sprintf("env:%s", env))
			} else {
				rd.SetId(fmt.Sprintf("file:%s", file))
			}

			// Look up the secret value from the environment variable
			// set by the provisioner at build time.
			var value string
			if env != "" {
				value = helpers.OptionalEnv(SecretEnvEnvironmentVariable(env))
			} else {
				value = helpers.OptionalEnv(SecretFileEnvironmentVariable(file))
			}

			// Only enforce missing secrets when we are certain this is a
			// workspace start build. We check both conditions:
			//  1. CODER_WORKSPACE_BUILD_ID is set (real build, not local
			//     terraform plan)
			//  2. CODER_WORKSPACE_TRANSITION is "start"
			// In all other cases (stop, delete, local dev, ambiguous state)
			// we return an empty value so the operation can proceed. This
			// prevents a missing or deleted secret from making a workspace
			// unstoppable or undeletable.
			if value == "" {
				buildID := os.Getenv("CODER_WORKSPACE_BUILD_ID")
				transition := os.Getenv("CODER_WORKSPACE_TRANSITION")
				if buildID != "" && transition == "start" {
					helpMessage := rd.Get("help_message").(string)
					return diag.Diagnostics{{
						Severity: diag.Error,
						Summary:  "Missing required secret",
						Detail:   helpMessage,
					}}
				}
			}
			_ = rd.Set("value", value)

			return nil
		},
		Schema: map[string]*schema.Schema{
			"env": {
				Type:        schema.TypeString,
				Description: "The environment variable name that this secret must inject (e.g. \"GITHUB_TOKEN\"). Must be POSIX-compliant: start with a letter or underscore, followed by letters, digits, or underscores. Exactly one of `env` or `file` must be set.",
				Optional:    true,
				ForceNew:    true,
			},
			"file": {
				Type:        schema.TypeString,
				Description: "The file path that this secret must inject (e.g. \"~/.aws/credentials\"). Must start with `~/` or `/`. Exactly one of `env` or `file` must be set.",
				Optional:    true,
				ForceNew:    true,
			},
			"help_message": {
				Type:        schema.TypeString,
				Description: "Guidance shown to users when this secret requirement is not satisfied. Displayed on the create workspace page and in build failure logs.",
				Required:    true,
			},
			"value": {
				Type:        schema.TypeString,
				Description: "The resolved secret value, populated from the user's stored secrets during workspace builds.",
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
}

// SecretEnvEnvironmentVariable returns the environment variable used
// to pass a user secret matched by env_name to Terraform during
// workspace builds. The env name is used directly and assumed to be
// POSIX-compliant.
func SecretEnvEnvironmentVariable(envName string) string {
	return fmt.Sprintf("CODER_SECRET_ENV_%s", envName)
}

// SecretFileEnvironmentVariable returns the environment variable used
// to pass a user secret matched by file_path to Terraform during
// workspace builds. The file path is hex-encoded because it contains
// characters invalid in environment variable names.
func SecretFileEnvironmentVariable(filePath string) string {
	return fmt.Sprintf("CODER_SECRET_FILE_%s", hex.EncodeToString([]byte(filePath)))
}
