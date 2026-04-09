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
			// Assert the full labeled-section format so refactors that
			// drop the summary, the "Required:" paragraph, the echoed
			// help_message, or the "To resolve:" action are caught.
			// The last line uses \s+ instead of a literal space because
			// Terraform soft-wraps long diagnostic lines at ~76 cols.
			ExpectError: regexp.MustCompile(
				`Missing required secret: environment variable "DOES_NOT_EXIST"[\s\S]*` +
					`Required: environment variable "DOES_NOT_EXIST"[\s\S]*` +
					`Help message: Please add the DOES_NOT_EXIST secret[\s\S]*` +
					`To resolve: ensure a secret exposes the environment\s+variable\s+"DOES_NOT_EXIST"`,
			),
		}},
	})
}

// nolint:paralleltest // t.Setenv is incompatible with t.Parallel.
func TestSecretMissingOnStartFile(t *testing.T) {
	// Missing file-path secret on start should fail with a file-flavored
	// diagnostic. Mirrors TestSecretMissingOnStart but covers the `file`
	// branch of the requirement builder.
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
				file         = "~/.missing/secret"
				help_message = "Please add the ~/.missing/secret secret"
			}
			`,
			ExpectError: regexp.MustCompile(
				`Missing required secret: file "~/.missing/secret"[\s\S]*` +
					`Required: file "~/.missing/secret"[\s\S]*` +
					`Help message: Please add the ~/.missing/secret secret[\s\S]*` +
					`To resolve: ensure a secret exposes the file "~/.missing/secret"`,
			),
		}},
	})
}

// nolint:paralleltest // t.Setenv is incompatible with t.Parallel.
func TestSecretMissingOnStartEmptyHelp(t *testing.T) {
	// When help_message is empty the diagnostic should omit the
	// "Help message:" paragraph entirely rather than render a blank one.
	// help_message is schema-required but HCL validates presence, not
	// non-emptiness, so `help_message = ""` is legal and must be handled.
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
				help_message = ""
			}
			`,
			// Require that "To resolve:" immediately follows "Required:"
			// with only blank lines between — no "Help message:" line.
			// Go's regexp lacks lookaheads, so this adjacency check is
			// how we assert absence.
			ExpectError: regexp.MustCompile(
				`Required: environment variable "DOES_NOT_EXIST"\s*\n\s*\n\s*To resolve:`,
			),
		}},
	})
}

// nolint:paralleltest // t.Setenv is incompatible with t.Parallel.
func TestSecretMissingOnLocalPlan(t *testing.T) {
	// A local `terraform plan` without a workspace build id must not
	// hard-fail on a missing secret. Only real workspace start builds
	// (transition == "start" AND CODER_WORKSPACE_BUILD_ID set) should.
	t.Setenv("CODER_WORKSPACE_TRANSITION", "start")
	// Explicitly clear BUILD_ID in case the surrounding environment
	// has it set.
	t.Setenv("CODER_WORKSPACE_BUILD_ID", "")
	resource.Test(t, resource.TestCase{
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
		Steps: []resource.TestStep{{
			Config: `
			provider "coder" {
			}
			data "coder_secret" "missing" {
				env          = "DOES_NOT_EXIST"
				help_message = "irrelevant"
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

// nolint:paralleltest // t.Setenv is incompatible with t.Parallel.
func TestSecretNeitherEnvNorFile(t *testing.T) {
	// Both `env` and `file` are optional in schema but the ReadContext
	// enforces that exactly one must be set. Covers the `env == "" &&
	// file == ""` branch.
	t.Setenv("CODER_WORKSPACE_TRANSITION", "stop")
	resource.Test(t, resource.TestCase{
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
		Steps: []resource.TestStep{{
			Config: `
			provider "coder" {
			}
			data "coder_secret" "neither" {
				help_message = "Pick one"
			}
			`,
			ExpectError: regexp.MustCompile("exactly one of `env` or `file` must be set"),
		}},
	})
}

// nolint:paralleltest // t.Setenv is incompatible with t.Parallel.
func TestSecretEnvInvalid(t *testing.T) {
	// Schema-level validation rejects non-POSIX env names at plan time,
	// before ReadContext runs. Each subtest pins one specific rejected
	// shape (leading digit, hyphen, space, dot) to guard against the
	// regex drifting in a way that silently accepts one of these.
	t.Setenv("CODER_WORKSPACE_TRANSITION", "stop")
	cases := []struct {
		name  string
		value string
	}{
		{name: "LeadingDigit", value: "1TOKEN"},
		{name: "Hyphen", value: "MY-TOKEN"},
		{name: "Space", value: "MY TOKEN"},
		{name: "Dot", value: "MY.TOKEN"},
		{name: "LeadingHyphen", value: "-TOKEN"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProviderFactories: coderFactory(),
				IsUnitTest:        true,
				Steps: []resource.TestStep{{
					Config: fmt.Sprintf(`
					provider "coder" {
					}
					data "coder_secret" "invalid" {
						env          = %q
						help_message = "ignored"
					}
					`, tc.value),
					// "POSIX-compliant" is the only substring guaranteed to
					// appear on the same rendered line as the error
					// headline. Terraform soft-wraps diagnostics at ~76
					// cols, which can split the regex source and the value.
					ExpectError: regexp.MustCompile("POSIX-compliant"),
				}},
			})
		})
	}
}

// nolint:paralleltest // t.Setenv is incompatible with t.Parallel.
func TestSecretEnvValid(t *testing.T) {
	// POSIX-valid env names must pass the validator. The happy path is
	// already covered by TestSecretByEnv; these cases exercise the
	// less-common shapes the regex permits (leading underscore, digits
	// after the first character, all-lowercase) to guard against the
	// validator being tightened by accident.
	t.Setenv("CODER_WORKSPACE_TRANSITION", "stop")
	cases := []struct {
		name  string
		value string
	}{
		{name: "LeadingUnderscore", value: "_TOKEN"},
		{name: "TrailingDigit", value: "TOKEN1"},
		{name: "AllLowercase", value: "my_token"},
		{name: "MixedCase", value: "MyToken"},
		{name: "SingleLetter", value: "X"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProviderFactories: coderFactory(),
				IsUnitTest:        true,
				Steps: []resource.TestStep{{
					Config: fmt.Sprintf(`
					provider "coder" {
					}
					data "coder_secret" "valid" {
						env          = %q
						help_message = "ignored"
					}
					`, tc.value),
				}},
			})
		})
	}
}

// nolint:paralleltest // t.Setenv is incompatible with t.Parallel.
func TestSecretFileInvalid(t *testing.T) {
	// Schema-level validation rejects non-absolute file paths at plan
	// time. The provisioner writes secrets using the path verbatim
	// (after `~/` expansion), so a relative path would land somewhere
	// surprising in the agent filesystem.
	t.Setenv("CODER_WORKSPACE_TRANSITION", "stop")
	cases := []struct {
		name  string
		value string
	}{
		{name: "Relative", value: "creds.txt"},
		{name: "RelativeDir", value: "config/creds"},
		{name: "DotRelative", value: "./creds"},
		{name: "ParentRelative", value: "../creds"},
		{name: "BareTilde", value: "~creds"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProviderFactories: coderFactory(),
				IsUnitTest:        true,
				Steps: []resource.TestStep{{
					Config: fmt.Sprintf(`
					provider "coder" {
					}
					data "coder_secret" "invalid" {
						file         = %q
						help_message = "ignored"
					}
					`, tc.value),
					// Soft-wrapping by Terraform can split across lines, so
					// match the stable prefix only.
					ExpectError: regexp.MustCompile("`file` must start with"),
				}},
			})
		})
	}
}

// nolint:paralleltest // t.Setenv is incompatible with t.Parallel.
func TestSecretFileValid(t *testing.T) {
	// Absolute and home-relative paths must pass the validator. Covers
	// shapes beyond the `~/.aws/credentials` case in TestSecretByFile.
	t.Setenv("CODER_WORKSPACE_TRANSITION", "stop")
	cases := []struct {
		name  string
		value string
	}{
		{name: "HomeDotfile", value: "~/.netrc"},
		{name: "HomeNested", value: "~/config/app/secret"},
		{name: "AbsoluteRoot", value: "/etc/creds"},
		{name: "AbsoluteNested", value: "/var/lib/secrets/token"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProviderFactories: coderFactory(),
				IsUnitTest:        true,
				Steps: []resource.TestStep{{
					Config: fmt.Sprintf(`
					provider "coder" {
					}
					data "coder_secret" "valid" {
						file         = %q
						help_message = "ignored"
					}
					`, tc.value),
				}},
			})
		})
	}
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
