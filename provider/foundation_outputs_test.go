package provider_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5/tf5server"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/stretchr/testify/require"
)

const foundationValues = `{"set":["a","b"],"list":[1,2],"map":{"a":true},"null_list":null,"nested":{"labels":{"a":"b"},"optional":null},"tuple":[true,"two"],"large":9007199254740993,"decimal":0.1234567890123456789,"secret":"private-marker"}`
const foundationMetadata = `{"set":{"type":["set","string"],"sensitive":false},"list":{"type":["list","number"],"sensitive":false},"map":{"type":["map","bool"],"sensitive":false},"null_list":{"type":["list","string"],"sensitive":false},"nested":{"type":["object",{"labels":["map","string"],"optional":"number"}],"sensitive":false},"tuple":{"type":["tuple",["bool","string"]],"sensitive":false},"large":{"type":"number","sensitive":false},"decimal":{"type":"number","sensitive":false},"secret":{"type":"string","sensitive":true}}`

func TestFoundationOutputsProtocol(t *testing.T) {
	for _, tc := range []struct {
		name, values, metadata string
	}{
		{name: "Mixed", values: foundationValues, metadata: foundationMetadata},
		{name: "AllPublic", values: foundationValues, metadata: strings.ReplaceAll(foundationMetadata, `"sensitive":true`, `"sensitive":false`)},
		{name: "AllSensitive", values: foundationValues, metadata: strings.ReplaceAll(foundationMetadata, `"sensitive":false`, `"sensitive":true`)},
		{name: "Empty", values: `{}`, metadata: `{}`},
		{name: "Unset"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clearFoundationEnv(t)
			t.Setenv("CODER_TEMPLATE_OUTPUTS", tc.values)
			t.Setenv("CODER_FOUNDATION_OUTPUT_METADATA", tc.metadata)
			server, sch := foundationServer(t)
			require.Len(t, sch.Block.Attributes, 6)
			for _, attr := range sch.Block.Attributes {
				require.Equal(t, attr.Name == "sensitive_outputs", attr.Sensitive)
				require.True(t, attr.Computed)
				require.False(t, attr.Optional)
				require.False(t, attr.Required)
				if attr.Name == "outputs" || attr.Name == "sensitive_outputs" {
					require.True(t, attr.Type.Is(tftypes.DynamicPseudoType))
				}
			}
			expectedType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"set":       tftypes.Set{ElementType: tftypes.String},
				"list":      tftypes.List{ElementType: tftypes.Number},
				"map":       tftypes.Map{ElementType: tftypes.Bool},
				"null_list": tftypes.List{ElementType: tftypes.String},
				"nested":    tftypes.Object{AttributeTypes: map[string]tftypes.Type{"labels": tftypes.Map{ElementType: tftypes.String}, "optional": tftypes.Number}},
				"tuple":     tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.Bool, tftypes.String}},
				"large":     tftypes.Number,
				"decimal":   tftypes.Number,
				"secret":    tftypes.String,
			}}
			expected, err := tftypes.ValueFromJSON([]byte(foundationValues), expectedType)
			require.NoError(t, err)
			var expectedOutputs map[string]tftypes.Value
			require.NoError(t, expected.As(&expectedOutputs))
			publicTypes, sensitiveTypes := make(map[string]tftypes.Type), make(map[string]tftypes.Type)
			publicValues, sensitiveValues := make(map[string]tftypes.Value), make(map[string]tftypes.Value)
			if tc.name != "Empty" && tc.name != "Unset" {
				for name, value := range expectedOutputs {
					if tc.name == "AllSensitive" || (tc.name == "Mixed" && name == "secret") {
						sensitiveTypes[name], sensitiveValues[name] = value.Type(), value
					} else {
						publicTypes[name], publicValues[name] = value.Type(), value
					}
				}
			}
			for range 2 {
				values := templateOutputValues(t, readFoundation(t, server, sch), sch)
				require.True(t, tftypes.NewValue(tftypes.Object{AttributeTypes: publicTypes}, publicValues).Equal(values["outputs"]))
				require.True(t, tftypes.NewValue(tftypes.Object{AttributeTypes: sensitiveTypes}, sensitiveValues).Equal(values["sensitive_outputs"]))
				if tc.name != "AllPublic" {
					require.NotContains(t, values["outputs"].String(), "private-marker")
					require.NotContains(t, values["outputs"].String(), "secret")
				}
			}
		})
	}
}

func TestFoundationOutputsLegacySnapshot(t *testing.T) {
	clearFoundationEnv(t)
	t.Setenv("CODER_TEMPLATE_OUTPUTS", foundationValues)
	t.Setenv("CODER_FOUNDATION_OUTPUT_METADATA", foundationMetadata)
	server, sch := foundationServer(t)
	foundation := templateOutputValues(t, readFoundation(t, server, sch), sch)
	schemas, err := server.GetProviderSchema(t.Context(), &tfprotov5.GetProviderSchemaRequest{})
	require.NoError(t, err)
	legacySchema := schemas.DataSourceSchemas["coder_template_outputs"]
	for _, attr := range legacySchema.Block.Attributes {
		if attr.Name == "outputs" {
			require.True(t, attr.Sensitive)
		}
	}
	legacy := templateOutputValues(t, readTemplateOutputs(t, server, legacySchema), legacySchema)
	var combined map[string]tftypes.Value
	require.NoError(t, legacy["outputs"].As(&combined))
	require.Len(t, combined, 9)
	for _, bucket := range []string{"outputs", "sensitive_outputs"} {
		var values map[string]tftypes.Value
		require.NoError(t, foundation[bucket].As(&values))
		for name, value := range values {
			require.True(t, value.Equal(combined[name]), "legacy output %s must retain its value and type", name)
		}
	}
}

func TestFoundationOutputsInvalid(t *testing.T) {
	for _, tc := range []struct {
		name, values, metadata, wantError string
	}{
		{name: "NoMetadata", values: `{"a":"private-marker"}`, wantError: "Foundation output metadata must describe exactly the outputs in CODER_TEMPLATE_OUTPUTS"},
		{name: "MissingKey", values: `{"a":"private-marker"}`, metadata: `{"b":{"type":"string","sensitive":false}}`, wantError: "Foundation output metadata must describe exactly the outputs in CODER_TEMPLATE_OUTPUTS"},
		{name: "IncompleteMixed", values: `{"public":true,"secret":"private-marker"}`, metadata: `{"public":{"type":"bool","sensitive":false}}`, wantError: "Foundation output metadata must describe exactly the outputs in CODER_TEMPLATE_OUTPUTS"},
		{name: "ExtraMetadata", metadata: `{"a":{"type":"string","sensitive":false}}`, wantError: "Foundation output metadata must describe exactly the outputs in CODER_TEMPLATE_OUTPUTS"},
		{name: "NullMetadata", metadata: `null`, wantError: "Foundation output metadata in CODER_FOUNDATION_OUTPUT_METADATA must be a JSON object"},
		{name: "ArrayMetadata", metadata: `[]`, wantError: "Foundation output metadata in CODER_FOUNDATION_OUTPUT_METADATA must be a JSON object"},
		{name: "TrailingMetadata", metadata: `{} {}`, wantError: "Foundation output metadata in CODER_FOUNDATION_OUTPUT_METADATA must be a JSON object"},
		{name: "DuplicateMetadata", values: `{"a":"private-marker"}`, metadata: `{"a":{"type":"string","sensitive":true},"a":{"type":"string","sensitive":false}}`, wantError: "Foundation output metadata in CODER_FOUNDATION_OUTPUT_METADATA must be a JSON object"},
		{name: "DuplicateSensitive", metadata: `{"a":{"type":"string","sensitive":true,"sensitive":false}}`, wantError: "Foundation output metadata must include a Terraform JSON type and a boolean sensitive flag for every output"},
		{name: "MissingSensitive", metadata: `{"a":{"type":"string"}}`, wantError: "Foundation output metadata must include a Terraform JSON type and a boolean sensitive flag for every output"},
		{name: "NullSensitive", metadata: `{"a":{"type":"string","sensitive":null}}`, wantError: "Foundation output metadata must include a Terraform JSON type and a boolean sensitive flag for every output"},
		{name: "StringSensitive", metadata: `{"a":{"type":"string","sensitive":"false"}}`, wantError: "Foundation output metadata must include a Terraform JSON type and a boolean sensitive flag for every output"},
		{name: "MissingType", metadata: `{"a":{"sensitive":false}}`, wantError: "Foundation output metadata must include a Terraform JSON type and a boolean sensitive flag for every output"},
		{name: "NullType", metadata: `{"a":{"type":null,"sensitive":false}}`, wantError: "Foundation output metadata must include a Terraform JSON type and a boolean sensitive flag for every output"},
		{name: "InvalidType", metadata: `{"a":{"type":"private-marker","sensitive":true}}`, wantError: "Foundation output metadata must include a Terraform JSON type and a boolean sensitive flag for every output"},
		{name: "NullEntry", metadata: `{"a":null}`, wantError: "Foundation output metadata must include a Terraform JSON type and a boolean sensitive flag for every output"},
		{name: "TypeMismatch", values: `{"a":"private-marker"}`, metadata: `{"a":{"type":["list","number"],"sensitive":true}}`, wantError: "Foundation output values do not match their Terraform types"},
		{name: "InvalidPublicBucket", values: `{"public":"private-marker","secret":"private-marker"}`, metadata: `{"public":{"type":["list","number"],"sensitive":false},"secret":{"type":"string","sensitive":true}}`, wantError: "Foundation output values do not match their Terraform types"},
		{name: "InvalidSensitiveBucket", values: `{"public":true,"secret":"private-marker"}`, metadata: `{"public":{"type":"bool","sensitive":false},"secret":{"type":["list","number"],"sensitive":true}}`, wantError: "Foundation output values do not match their Terraform types"},
		{name: "DuplicateValues", values: `{"a":"private-marker","a":"private-marker"}`, wantError: "Shared resource outputs in CODER_TEMPLATE_OUTPUTS must be a JSON object"},
		{name: "InvalidJSON", values: `{"a":"private-marker"`, wantError: "Shared resource outputs in CODER_TEMPLATE_OUTPUTS must be a JSON object"},
		{name: "NullValues", values: `null`, wantError: "Shared resource outputs in CODER_TEMPLATE_OUTPUTS must be a JSON object"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clearFoundationEnv(t)
			t.Setenv("CODER_TEMPLATE_OUTPUTS", tc.values)
			t.Setenv("CODER_FOUNDATION_OUTPUT_METADATA", tc.metadata)
			server, sch := foundationServer(t)
			resp := readFoundation(t, server, sch)
			require.Len(t, resp.Diagnostics, 1)
			require.Equal(t, tfprotov5.DiagnosticSeverityError, resp.Diagnostics[0].Severity)
			require.Equal(t, tc.wantError, resp.Diagnostics[0].Summary)
			require.Empty(t, resp.Diagnostics[0].Detail)
			require.NotContains(t, resp.Diagnostics[0].Summary, "private-marker")
			state, err := resp.State.Unmarshal(sch.ValueType())
			require.NoError(t, err)
			var attributes map[string]tftypes.Value
			require.NoError(t, state.As(&attributes))
			require.True(t, attributes["outputs"].IsNull(), "invalid snapshots must not publish public outputs")
			require.True(t, attributes["sensitive_outputs"].IsNull(), "invalid snapshots must not publish sensitive outputs")
			require.NotContains(t, state.String(), "private-marker")
		})
	}
}

func TestFoundationOutputsTerraform(t *testing.T) {
	clearFoundationEnv(t)
	t.Setenv("CODER_TEMPLATE_OUTPUTS", foundationValues)
	t.Setenv("CODER_FOUNDATION_OUTPUT_METADATA", foundationMetadata)
	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: coderFactory(),
		IsUnitTest:               true,
		Steps: []resource.TestStep{{
			Config: `
data "coder_foundation" "current" {}
locals { outputs = data.coder_foundation.current.outputs }
resource "terraform_data" "secret" {
  input = data.coder_foundation.current.sensitive_outputs.secret
}
output "types_preserved" {
  value = (
    local.outputs.set == toset(["a", "b"]) &&
    local.outputs.list == tolist([1, 2]) &&
    local.outputs.map == tomap({a = true}) &&
    local.outputs.null_list == tolist(null) &&
    local.outputs.nested.labels == tomap({a = "b"}) &&
    local.outputs.nested.optional == tonumber(null) &&
    local.outputs.tuple == [true, "two"] &&
    local.outputs.large == 9007199254740993 &&
    local.outputs.decimal == 0.1234567890123456789
  )
}
output "secret_consumed" {
  sensitive = true
  value = terraform_data.secret.output
}
output "sensitivity_preserved" { value = issensitive(data.coder_foundation.current.sensitive_outputs.secret) }
output "public_not_sensitive" { value = !issensitive(local.outputs) && !issensitive(local.outputs.set) }
output "no_leakage" { value = !can(local.outputs.secret) }
output "no_public_in_sensitive" { value = !can(data.coder_foundation.current.sensitive_outputs.set) }
output "optional" { value = try(local.outputs.missing, "fallback") }
`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckOutput("types_preserved", "true"),
				resource.TestCheckOutput("secret_consumed", "private-marker"),
				resource.TestCheckOutput("sensitivity_preserved", "true"),
				resource.TestCheckOutput("public_not_sensitive", "true"),
				resource.TestCheckOutput("no_leakage", "true"),
				resource.TestCheckOutput("no_public_in_sensitive", "true"),
				resource.TestCheckOutput("optional", "fallback"),
			),
		}},
	})
}

// The SDK's legacy state shim rejects string resource indexes. Exercise
// for_each with Terraform directly instead of passing state through that shim.
func TestFoundationOutputsTerraformForEach(t *testing.T) {
	clearFoundationEnv(t)
	t.Setenv("CODER_TEMPLATE_OUTPUTS", foundationValues)
	t.Setenv("CODER_FOUNDATION_OUTPUT_METADATA", foundationMetadata)
	binary := os.Getenv("TF_ACC_TERRAFORM_PATH")
	if binary == "" {
		var err error
		binary, err = exec.LookPath("terraform")
		require.NoError(t, err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	server, _ := foundationServer(t)
	reattach := make(chan *plugin.ReattachConfig, 1)
	closed := make(chan struct{})
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- tf5server.Serve("registry.terraform.io/coder/coder", func() tfprotov5.ProviderServer { return server },
			tf5server.WithDebug(ctx, reattach, closed), tf5server.WithLoggingSink(t), tf5server.WithoutLogStderrOverride())
	}()
	defer func() {
		cancel()
		select {
		case <-closed:
		case <-time.After(time.Minute):
			t.Error("provider did not shut down")
		}
	}()
	var config *plugin.ReattachConfig
	select {
	case config = <-reattach:
	case err := <-serveErr:
		t.Fatalf("provider exited before reattach: %v", err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	reattachJSON, err := json.Marshal(map[string]any{
		"registry.terraform.io/coder/coder": map[string]any{
			"Protocol": config.Protocol, "ProtocolVersion": config.ProtocolVersion,
			"Pid": config.Pid, "Test": config.Test,
			"Addr": map[string]string{"Network": config.Addr.Network(), "String": config.Addr.String()},
		},
	})
	require.NoError(t, err)
	dir := t.TempDir()
	// Isolate the test from installed development overrides and CLI settings.
	cliConfig := filepath.Join(dir, "terraform.rc")
	require.NoError(t, os.WriteFile(cliConfig, nil, 0600))
	t.Setenv("TF_CLI_CONFIG_FILE", cliConfig)
	t.Setenv("TF_REATTACH_PROVIDERS", string(reattachJSON))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf"), []byte(`
terraform {
  required_providers {
    coder = { source = "coder/coder" }
  }
}
data "coder_foundation" "current" {}
resource "terraform_data" "public" {
  for_each = data.coder_foundation.current.outputs.set
  input = each.value
}
resource "terraform_data" "secret" {
  input = data.coder_foundation.current.sensitive_outputs.secret
}
output "public_instances" { value = sort(keys(terraform_data.public)) }
output "secret" {
  sensitive = true
  value = terraform_data.secret.output
}
`), 0600))
	run := func(args ...string) []byte {
		t.Helper()
		cmd := exec.CommandContext(ctx, binary, args...)
		cmd.Dir = dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "%s", output)
		return output
	}
	run("init", "-input=false", "-no-color")
	plan := run("plan", "-input=false", "-no-color", "-out=plan.tfplan")
	require.Contains(t, string(plan), `terraform_data.public["a"]`)
	require.Contains(t, string(plan), `terraform_data.public["b"]`)
	require.NotContains(t, string(plan), "private-marker")
	applied := run("apply", "-input=false", "-no-color", "plan.tfplan")
	require.NotContains(t, string(applied), "private-marker")
	var outputs map[string]struct {
		Sensitive bool            `json:"sensitive"`
		Value     json.RawMessage `json:"value"`
	}
	require.NoError(t, json.Unmarshal(run("output", "-json"), &outputs))
	require.JSONEq(t, `["a","b"]`, string(outputs["public_instances"].Value))
	require.False(t, outputs["public_instances"].Sensitive)
	require.JSONEq(t, `"private-marker"`, string(outputs["secret"].Value))
	require.True(t, outputs["secret"].Sensitive)
}

func TestFoundationOutputsEmptyFallback(t *testing.T) {
	clearFoundationEnv(t)
	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: coderFactory(),
		IsUnitTest:               true,
		Steps: []resource.TestStep{{
			Config: `
data "coder_foundation" "current" {}
output "optional" { value = join(",", try(data.coder_foundation.current.outputs.network.subnets, ["fallback"])) }
output "empty" { value = data.coder_foundation.current.outputs == {} }
output "public_not_sensitive" { value = !issensitive(data.coder_foundation.current.outputs) }
output "empty_sensitive" {
  sensitive = true
  value = data.coder_foundation.current.sensitive_outputs == {}
}
output "always_sensitive" { value = issensitive(data.coder_foundation.current.sensitive_outputs) }
output "optional_secret" { value = try(data.coder_foundation.current.sensitive_outputs.missing, "fallback") }
`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckOutput("empty", "true"),
				resource.TestCheckOutput("public_not_sensitive", "true"),
				resource.TestCheckOutput("empty_sensitive", "true"),
				resource.TestCheckOutput("always_sensitive", "true"),
				resource.TestCheckOutput("optional", "fallback"),
				resource.TestCheckOutput("optional_secret", "fallback"),
			),
		}},
	})
}

func TestFoundationOutputsRequireSensitiveOutput(t *testing.T) {
	clearFoundationEnv(t)
	t.Setenv("CODER_TEMPLATE_OUTPUTS", `{"secret":"private-marker"}`)
	t.Setenv("CODER_FOUNDATION_OUTPUT_METADATA", `{"secret":{"type":"string","sensitive":true}}`)
	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: coderFactory(),
		IsUnitTest:               true,
		Steps: []resource.TestStep{{
			Config: `
data "coder_foundation" "current" {}
output "secret" { value = data.coder_foundation.current.sensitive_outputs.secret }
`,
			ExpectError: regexp.MustCompile("Output refers to sensitive values"),
		}},
	})
}
