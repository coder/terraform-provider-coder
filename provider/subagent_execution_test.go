package provider_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"

	"github.com/coder/terraform-provider-coder/v2/provider"
)

func TestSubagentExecution(t *testing.T) {
	t.Parallel()

	t.Run("Schema", func(t *testing.T) {
		t.Parallel()

		resourceSchema := provider.New().ResourcesMap["coder_subagent_execution"]
		require.NotNil(t, resourceSchema)
		require.Equal(t, 1, resourceSchema.SchemaVersion)
		require.NotNil(t, resourceSchema.CreateContext)
		require.NotNil(t, resourceSchema.ReadContext)
		require.NotNil(t, resourceSchema.DeleteContext)
		require.Nil(t, resourceSchema.UpdateContext)
		require.Nil(t, resourceSchema.Importer)
		require.Empty(t, resourceSchema.StateUpgraders)
		require.NotContains(t, resourceSchema.Schema, "id")

		for _, field := range []string{
			"agent_id",
			"name",
			"driver",
			"driver_protocol",
			"shared_host_path",
			"shared_child_path",
			"startup_timeout",
			"restart_policy",
		} {
			require.True(t, resourceSchema.Schema[field].ForceNew, "%s must be ForceNew", field)
		}
	})

	t.Run("MinimalConfigDefaultsAndIDs", func(t *testing.T) {
		t.Parallel()

		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
			Steps: []resource.TestStep{{
				Config: subagentExecutionConfig(),
				Check: func(state *terraform.State) error {
					execution := state.Modules[0].Resources["coder_subagent_execution.example"]
					require.NotNil(t, execution)
					require.Equal(t, "parent-agent", execution.Primary.Attributes["agent_id"])
					require.Equal(t, "child", execution.Primary.Attributes["name"])
					require.Equal(t, "example-driver", execution.Primary.Attributes["driver"])
					require.Equal(t, "1", execution.Primary.Attributes["driver_protocol"])
					require.Equal(t, "host/path", execution.Primary.Attributes["shared_host_path"])
					require.Equal(t, "child/path", execution.Primary.Attributes["shared_child_path"])
					require.Equal(t, "120", execution.Primary.Attributes["startup_timeout"])
					require.Equal(t, "on-failure", execution.Primary.Attributes["restart_policy"])

					resourceID := execution.Primary.ID
					subagentID := execution.Primary.Attributes["subagent_id"]
					_, err := uuid.Parse(resourceID)
					require.NoError(t, err, "resource ID should be a valid UUID")
					_, err = uuid.Parse(subagentID)
					require.NoError(t, err, "subagent_id should be a valid UUID")
					require.NotEqual(t, resourceID, subagentID)
					return nil
				},
			}},
		})
	})

	t.Run("FullConfig", func(t *testing.T) {
		t.Parallel()

		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
			Steps: []resource.TestStep{{
				Config: subagentExecutionConfig(
					`name = "child"`, `name = "Child-01"`,
					`driver = "example-driver"`, `driver = "full driver configuration"`,
					`shared_host_path = "host/path"`, `shared_host_path = "relative-host-path"`,
					`shared_child_path = "child/path"`, `shared_child_path = "relative-child-path"
					driver_protocol = 1
					startup_timeout = 2147483647
					restart_policy = "never"`,
				),
				Check: func(state *terraform.State) error {
					execution := state.Modules[0].Resources["coder_subagent_execution.example"]
					require.NotNil(t, execution)
					for key, expected := range map[string]string{
						"agent_id":          "parent-agent",
						"name":              "Child-01",
						"driver":            "full driver configuration",
						"driver_protocol":   "1",
						"shared_host_path":  "relative-host-path",
						"shared_child_path": "relative-child-path",
						"startup_timeout":   "2147483647",
						"restart_policy":    "never",
					} {
						require.Equal(t, expected, execution.Primary.Attributes[key], key)
					}
					return nil
				},
			}},
		})
	})

	invalidCases := []struct {
		name        string
		oldValue    string
		newValue    string
		expectError *regexp.Regexp
	}{
		{
			name:        "InvalidName",
			oldValue:    `name = "child"`,
			newValue:    `name = "bad--name"`,
			expectError: regexp.MustCompile(`invalid value for name`),
		},
		{
			name:        "BlankDriver",
			oldValue:    `driver = "example-driver"`,
			newValue:    `driver = "   "`,
			expectError: regexp.MustCompile(`expected "driver" to not be an empty string or whitespace`),
		},
		{
			name:        "BlankSharedHostPath",
			oldValue:    `shared_host_path = "host/path"`,
			newValue:    `shared_host_path = "\t"`,
			expectError: regexp.MustCompile(`expected "shared_host_path" to not be an empty string or whitespace`),
		},
		{
			name:        "BlankSharedChildPath",
			oldValue:    `shared_child_path = "child/path"`,
			newValue:    `shared_child_path = "\n"`,
			expectError: regexp.MustCompile(`expected "shared_child_path" to not be an empty string or whitespace`),
		},
		{
			name:     "UnsupportedProtocol",
			oldValue: `shared_child_path = "child/path"`,
			newValue: `shared_child_path = "child/path"
			driver_protocol = 2`,
			expectError: regexp.MustCompile(`expected driver_protocol to be one of`),
		},
		{
			name:     "TimeoutTooLow",
			oldValue: `shared_child_path = "child/path"`,
			newValue: `shared_child_path = "child/path"
			startup_timeout = 0`,
			expectError: regexp.MustCompile(`expected startup_timeout to be in the range`),
		},
		{
			name:     "TimeoutTooHigh",
			oldValue: `shared_child_path = "child/path"`,
			newValue: `shared_child_path = "child/path"
			startup_timeout = 2147483648`,
			expectError: regexp.MustCompile(`expected startup_timeout to be in the range`),
		},
		{
			name:     "InvalidRestartPolicy",
			oldValue: `shared_child_path = "child/path"`,
			newValue: `shared_child_path = "child/path"
			restart_policy = "always"`,
			expectError: regexp.MustCompile(`expected restart_policy to be one of`),
		},
	}

	for _, testCase := range invalidCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			resource.Test(t, resource.TestCase{
				ProviderFactories: coderFactory(),
				IsUnitTest:        true,
				Steps: []resource.TestStep{{
					Config:      subagentExecutionConfig(testCase.oldValue, testCase.newValue),
					ExpectError: testCase.expectError,
				}},
			})
		})
	}
}

func TestSubagentExecutionReferences(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
		Steps: []resource.TestStep{{
			Config: `
				provider "coder" {}

				resource "coder_agent" "parent" {
					os = "linux"
					arch = "amd64"
				}

				resource "coder_subagent_execution" "child" {
					agent_id = coder_agent.parent.id
					name = "child"
					driver = "example-driver"
					shared_host_path = "/workspace"
					shared_child_path = "/workspace"
				}

				resource "coder_app" "child" {
					agent_id = coder_subagent_execution.child.subagent_id
					slug = "child-app"
					command = "bash"
				}

				resource "coder_script" "child" {
					agent_id = coder_subagent_execution.child.subagent_id
					display_name = "Child setup"
					run_on_start = true
					script = "echo ready"
				}

				resource "coder_env" "child" {
					agent_id = coder_subagent_execution.child.subagent_id
					name = "CHILD_EXECUTION"
					value = "true"
				}
			`,
			Check: func(state *terraform.State) error {
				execution := state.Modules[0].Resources["coder_subagent_execution.child"]
				require.NotNil(t, execution)
				subagentID := execution.Primary.Attributes["subagent_id"]
				_, err := uuid.Parse(subagentID)
				require.NoError(t, err)

				for _, address := range []string{
					"coder_app.child",
					"coder_script.child",
					"coder_env.child",
				} {
					childResource := state.Modules[0].Resources[address]
					require.NotNil(t, childResource, address)
					require.Equal(t, subagentID, childResource.Primary.Attributes["agent_id"], address)
				}
				return nil
			},
		}},
	})
}

func subagentExecutionConfig(replacements ...string) string {
	config := `
		provider "coder" {}

		resource "coder_subagent_execution" "example" {
			agent_id = "parent-agent"
			name = "child"
			driver = "example-driver"
			shared_host_path = "host/path"
			shared_child_path = "child/path"
		}
	`
	if len(replacements)%2 != 0 {
		panic("subagent execution config replacements must be pairs")
	}
	for i := 0; i < len(replacements); i += 2 {
		if !strings.Contains(config, replacements[i]) {
			panic("subagent execution config replacement target not found: " + replacements[i])
		}
		config = strings.Replace(config, replacements[i], replacements[i+1], 1)
	}
	return config
}
