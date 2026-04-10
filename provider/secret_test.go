package provider_test

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"

	"github.com/coder/terraform-provider-coder/v2/provider"
)

// nolint:paralleltest // t.Setenv is incompatible with t.Parallel.
func TestSecretByEnv(t *testing.T) {

	t.Setenv("CODER_WORKSPACE_TRANSITION", "stop")
	resource.Test(t, resource.TestCase{
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
		Steps: []resource.TestStep{{
			Config: `
			provider "coder" {
			}
			data "coder_secret" "github_token" {
				env          = "GITHUB_TOKEN"
				help_message = "Add a GitHub PAT as a secret with env=GITHUB_TOKEN"
			}
			`,
			Check: func(state *terraform.State) error {
				require.Len(t, state.Modules, 1)
				require.Len(t, state.Modules[0].Resources, 1)
				res := state.Modules[0].Resources["data.coder_secret.github_token"]
				require.NotNil(t, res)

				attribs := res.Primary.Attributes
				require.Equal(t, "env:GITHUB_TOKEN", attribs["id"])
				require.Equal(t, "GITHUB_TOKEN", attribs["env"])
				require.Equal(t, "", attribs["value"])

				return nil
			},
		}},
	})
}

// nolint:paralleltest // t.Setenv is incompatible with t.Parallel.
func TestSecretByFile(t *testing.T) {

	t.Setenv("CODER_WORKSPACE_TRANSITION", "stop")
	resource.Test(t, resource.TestCase{
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
		Steps: []resource.TestStep{{
			Config: `
			provider "coder" {
			}
			data "coder_secret" "aws_creds" {
				file         = "~/.aws/credentials"
				help_message = "Add your AWS credentials file as a secret"
			}
			`,
			Check: func(state *terraform.State) error {
				require.Len(t, state.Modules, 1)
				require.Len(t, state.Modules[0].Resources, 1)
				res := state.Modules[0].Resources["data.coder_secret.aws_creds"]
				require.NotNil(t, res)

				attribs := res.Primary.Attributes
				require.Equal(t, "file:~/.aws/credentials", attribs["id"])
				require.Equal(t, "~/.aws/credentials", attribs["file"])
				require.Equal(t, "", attribs["value"])

				return nil
			},
		}},
	})
}

// nolint:paralleltest // t.Setenv is incompatible with t.Parallel.
func TestSecretWithEnvValue(t *testing.T) {

	t.Setenv(provider.SecretEnvEnvironmentVariable("MY_TOKEN"), "secret-token-value")
	resource.Test(t, resource.TestCase{
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
		Steps: []resource.TestStep{{
			Config: `
			provider "coder" {
			}
			data "coder_secret" "my_token" {
				env          = "MY_TOKEN"
				help_message = "Set the MY_TOKEN secret"
			}
			`,
			Check: func(state *terraform.State) error {
				require.Len(t, state.Modules, 1)
				require.Len(t, state.Modules[0].Resources, 1)
				res := state.Modules[0].Resources["data.coder_secret.my_token"]
				require.NotNil(t, res)

				attribs := res.Primary.Attributes
				require.Equal(t, "secret-token-value", attribs["value"])

				return nil
			},
		}},
	})
}

// nolint:paralleltest // t.Setenv is incompatible with t.Parallel.
func TestSecretWithFileValue(t *testing.T) {

	t.Setenv(provider.SecretFileEnvironmentVariable("~/.ssh/id_rsa"), "private-key-contents")
	resource.Test(t, resource.TestCase{
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
		Steps: []resource.TestStep{{
			Config: `
			provider "coder" {
			}
			data "coder_secret" "ssh_key" {
				file         = "~/.ssh/id_rsa"
				help_message = "Add your SSH private key"
			}
			`,
			Check: func(state *terraform.State) error {
				require.Len(t, state.Modules, 1)
				require.Len(t, state.Modules[0].Resources, 1)
				res := state.Modules[0].Resources["data.coder_secret.ssh_key"]
				require.NotNil(t, res)

				attribs := res.Primary.Attributes
				require.Equal(t, "private-key-contents", attribs["value"])

				return nil
			},
		}},
	})
}

// nolint:paralleltest // t.Setenv is incompatible with t.Parallel.
func TestSecretMissingOnStart(t *testing.T) {

	// Default transition is "start", and no env var is set for the
	// secret, so the data source should fail.
	t.Setenv("CODER_WORKSPACE_TRANSITION", "start")
	t.Setenv("CODER_WORKSPACE_BUILD_ID", "test-build-id")
	resource.Test(t, resource.TestCase{
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
		Steps: []resource.TestStep{{
			Config: `
			provider "coder" {
			}
			data "coder_secret" "missing" {
				env          = "DOES_NOT_EXIST"
				help_message = "Please add the DOES_NOT_EXIST secret"
			}
			`,
			ExpectError: regexp.MustCompile("Missing required secret"),
		}},
	})
}

// nolint:paralleltest // t.Setenv is incompatible with t.Parallel.
func TestSecretMissingOnStop(t *testing.T) {

	// On stop transitions, missing secrets should not error.
	t.Setenv("CODER_WORKSPACE_TRANSITION", "stop")
	resource.Test(t, resource.TestCase{
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
		Steps: []resource.TestStep{{
			Config: `
			provider "coder" {
			}
			data "coder_secret" "missing" {
				env          = "DOES_NOT_EXIST"
				help_message = "Please add the DOES_NOT_EXIST secret"
			}
			`,
			Check: func(state *terraform.State) error {
				res := state.Modules[0].Resources["data.coder_secret.missing"]
				require.NotNil(t, res)
				require.Equal(t, "", res.Primary.Attributes["value"])
				return nil
			},
		}},
	})
}

// nolint:paralleltest // t.Setenv is incompatible with t.Parallel.
func TestSecretBothEnvAndFile(t *testing.T) {

	t.Setenv("CODER_WORKSPACE_TRANSITION", "stop")
	resource.Test(t, resource.TestCase{
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
		Steps: []resource.TestStep{{
			Config: `
			provider "coder" {
			}
			data "coder_secret" "both" {
				env          = "MY_SECRET"
				file         = "~/.my-secret"
				help_message = "Pick one"
			}
			`,
			ExpectError: regexp.MustCompile("exactly one of `env` or `file` must be set"),
		}},
	})
}

func TestSecretEnvironmentVariables(t *testing.T) {
	t.Parallel()

	t.Run("EnvSecret", func(t *testing.T) {
		t.Parallel()
		result := provider.SecretEnvEnvironmentVariable("GITHUB_TOKEN")
		require.Equal(t, "CODER_SECRET_ENV_GITHUB_TOKEN", result)
	})

	t.Run("FileSecret", func(t *testing.T) {
		t.Parallel()
		filePath := "~/.aws/credentials"
		result := provider.SecretFileEnvironmentVariable(filePath)
		expected := fmt.Sprintf("CODER_SECRET_FILE_%s", hex.EncodeToString([]byte(filePath)))
		require.Equal(t, expected, result)
	})
}
