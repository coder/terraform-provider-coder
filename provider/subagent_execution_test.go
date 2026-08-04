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

		expectedFields := []struct {
			name       string
			required   bool
			optional   bool
			computed   bool
			forceNew   bool
			defaultVal interface{}
		}{
			{name: "agent_id", required: true, forceNew: true},
			{name: "name", required: true, forceNew: true},
			{name: "driver", required: true, forceNew: true},
			{name: "driver_protocol", optional: true, forceNew: true, defaultVal: 1},
			{name: "shared_host_path", required: true, forceNew: true},
			{name: "shared_child_path", required: true, forceNew: true},
			{name: "startup_timeout", optional: true, forceNew: true, defaultVal: 120},
			{name: "restart_policy", optional: true, forceNew: true, defaultVal: "on-failure"},
			{name: "subagent_id", computed: true},
		}
		require.Len(t, resourceSchema.Schema, len(expectedFields))
		for _, expected := range expectedFields {
			field, ok := resourceSchema.Schema[expected.name]
			require.True(t, ok, "%s must be present", expected.name)
			require.Equal(t, expected.required, field.Required, "%s Required", expected.name)
			require.Equal(t, expected.optional, field.Optional, "%s Optional", expected.name)
			require.Equal(t, expected.computed, field.Computed, "%s Computed", expected.name)
			require.Equal(t, expected.forceNew, field.ForceNew, "%s ForceNew", expected.name)
			require.Equal(t, expected.defaultVal, field.Default, "%s Default", expected.name)
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

	const config = `
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
	`
	replacementConfig := strings.Replace(config, `driver = "example-driver"`, `driver = "replacement-driver"`, 1)

	var initialResourceID, initialSubagentID string
	requireDownstreamAgentIDs := func(state *terraform.State, expectedAgentID string) {
		for _, address := range []string{
			"coder_app.child",
			"coder_script.child",
			"coder_env.child",
		} {
			childResource := state.Modules[0].Resources[address]
			require.NotNil(t, childResource, address)
			require.Equal(t, expectedAgentID, childResource.Primary.Attributes["agent_id"], address)
		}
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: func(state *terraform.State) error {
					execution := state.Modules[0].Resources["coder_subagent_execution.child"]
					require.NotNil(t, execution)

					initialResourceID = execution.Primary.ID
					initialSubagentID = execution.Primary.Attributes["subagent_id"]
					_, err := uuid.Parse(initialResourceID)
					require.NoError(t, err, "resource ID should be a valid UUID")
					_, err = uuid.Parse(initialSubagentID)
					require.NoError(t, err, "subagent_id should be a valid UUID")
					require.NotEqual(t, initialResourceID, initialSubagentID)
					requireDownstreamAgentIDs(state, initialSubagentID)
					return nil
				},
			},
			{
				Config: replacementConfig,
				Check: func(state *terraform.State) error {
					execution := state.Modules[0].Resources["coder_subagent_execution.child"]
					require.NotNil(t, execution)
					require.Equal(t, "replacement-driver", execution.Primary.Attributes["driver"])

					replacementResourceID := execution.Primary.ID
					replacementSubagentID := execution.Primary.Attributes["subagent_id"]
					_, err := uuid.Parse(replacementResourceID)
					require.NoError(t, err, "replacement resource ID should be a valid UUID")
					_, err = uuid.Parse(replacementSubagentID)
					require.NoError(t, err, "replacement subagent_id should be a valid UUID")
					require.NotEqual(t, initialResourceID, replacementResourceID, "resource ID must rotate on replacement")
					require.NotEqual(t, initialSubagentID, replacementSubagentID, "subagent_id must rotate on replacement")
					require.NotEqual(t, replacementResourceID, replacementSubagentID)
					requireDownstreamAgentIDs(state, replacementSubagentID)
					return nil
				},
			},
		},
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
