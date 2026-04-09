package provider

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/coder/terraform-provider-coder/v2/provider/helpers"
)

// posixEnvNameRegex matches a POSIX-compliant environment variable name:
// starts with a letter or underscore, followed by letters, digits, or
// underscores. This mirrors the rule enforced by coderd when secrets are
// created, so enforcing it in the provider catches typos at terraform
// validate/plan time rather than at build time.
var posixEnvNameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// validateSecretEnv rejects env names that can never match a stored secret.
// Empty values pass through: the env/file mutex check in ReadContext handles
// that case and produces a clearer error.
func validateSecretEnv(val any, _ cty.Path) diag.Diagnostics {
	s, ok := val.(string)
	if !ok {
		return diag.Errorf("expected string, got %T", val)
	}
	if s == "" {
		return nil
	}
	if !posixEnvNameRegex.MatchString(s) {
		return diag.Errorf(
			"`env` must be a POSIX-compliant identifier matching %q; got %q",
			posixEnvNameRegex.String(), s)
	}
	return nil
}

// validateSecretFile rejects file paths that are not absolute or home-relative.
// This mirrors the rule enforced by coderd when secrets are created/updated
// (paths must start with `~/` or `/`), so enforcing it in the provider catches
// mistakes at terraform validate/plan time rather than at build time.
func validateSecretFile(val any, _ cty.Path) diag.Diagnostics {
	s, ok := val.(string)
	if !ok {
		return diag.Errorf("expected string, got %T", val)
	}
	if s == "" {
		return nil
	}
	if !strings.HasPrefix(s, "/") && !strings.HasPrefix(s, "~/") {
		return diag.Errorf(
			"`file` must start with `/` or `~/`; got %q", s)
	}
	return nil
}

// secretDataSource returns a schema for a user secret data source.
func secretDataSource() *schema.Resource {
	const valueKey = "value"

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

			if value != "" {
				// Happy path where secret is resolved.
				_ = rd.Set(valueKey, value)
				return nil
			}

			// Note that an value is treated as missing. The provider cannot
			// distinguish "user has not stored the secret" from "user stored
			// an empty value", because both surface as an unset or empty
			// CODER_SECRET_* env var. This means a user must have a non-empty
			// secret value to satisfy a requirement.

			// Only enforce missing secrets when we are certain this is a
			// workspace start build. We check both conditions:
			//  1. CODER_WORKSPACE_BUILD_ID is set (real build, not local
			//     terraform plan)
			//  2. CODER_WORKSPACE_TRANSITION is "start"
			// In all other cases (stop, delete, local dev, ambiguous state)
			// we return an empty value so the operation can proceed. This
			// prevents a missing or deleted secret from making a workspace
			// unstoppable or undeletable.
			buildID := os.Getenv("CODER_WORKSPACE_BUILD_ID")
			transition := os.Getenv("CODER_WORKSPACE_TRANSITION")
			workspaceStartBuild := buildID != "" && transition == "start"
			if !workspaceStartBuild {
				_ = rd.Set(valueKey, value)
				return nil
			}

			var requirement string
			if env != "" {
				requirement = fmt.Sprintf("environment variable %q", env)
			} else {
				requirement = fmt.Sprintf("file %q", file)
			}

			var detail strings.Builder
			_, _ = fmt.Fprintf(&detail, "Required: %s\n\n", requirement)
			if helpMessage := rd.Get("help_message").(string); helpMessage != "" {
				_, _ = fmt.Fprintf(&detail, "Help message: %s\n\n", helpMessage)
			}
			_, _ = fmt.Fprintf(&detail, "To resolve: ensure a secret exposes the %s.\n", requirement)

			return diag.Diagnostics{{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("Missing required secret: %s", requirement),
				Detail:   detail.String(),
			}}
		},
		Schema: map[string]*schema.Schema{
			"env": {
				Type:             schema.TypeString,
				Description:      "The environment variable name that this secret must inject (e.g. \"MY_TOKEN\"). Must be POSIX-compliant: start with a letter or underscore, followed by letters, digits, or underscores. Exactly one of `env` or `file` must be set.",
				Optional:         true,
				ForceNew:         true,
				ValidateDiagFunc: validateSecretEnv,
			},
			"file": {
				Type:             schema.TypeString,
				Description:      "The file path that this secret must inject (e.g. \"~/my-token\"). Must start with `~/` or `/`. Exactly one of `env` or `file` must be set.",
				Optional:         true,
				ForceNew:         true,
				ValidateDiagFunc: validateSecretFile,
			},
			"help_message": {
				Type:        schema.TypeString,
				Description: "Guidance shown to users when this secret requirement is not satisfied. Displayed on the create workspace page and in build failure logs.",
				Required:    true,
			},
			"value": {
				Type:        schema.TypeString,
				Description: "The resolved secret value, populated from the user's stored secrets during workspace builds. Treated as missing if empty.",
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
