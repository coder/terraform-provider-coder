package provider_test

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"

	"github.com/coder/terraform-provider-coder/v2/provider"
)

func TestTemplateOutputsProtocol(t *testing.T) {
	server, err := provider.NewMux(t.Context())
	require.NoError(t, err)
	schemas, err := server.GetProviderSchema(t.Context(), &tfprotov5.GetProviderSchemaRequest{})
	require.NoError(t, err)
	require.Empty(t, schemas.Diagnostics)
	require.Contains(t, schemas.ResourceSchemas, "coder_agent")
	require.Contains(t, schemas.DataSourceSchemas, "coder_workspace")
	sdk, err := provider.New().GRPCProvider().GetProviderSchema(t.Context(), &tfprotov5.GetProviderSchemaRequest{})
	require.NoError(t, err)
	require.Equal(t, sdk.Provider, schemas.Provider)
	sch := schemas.DataSourceSchemas["coder_template_outputs"]
	require.Len(t, sch.Block.Attributes, 3)
	for _, attr := range sch.Block.Attributes {
		if attr.Name == "outputs" {
			require.True(t, attr.Type.Is(tftypes.DynamicPseudoType))
			require.True(t, attr.Computed)
		}
	}

	for _, tc := range []struct {
		name, outputs, revisionID, wantError string
	}{
		{name: "EmptyObject", outputs: "{}"},
		{name: "EmptyEnvironment"},
		{name: "MalformedJSON", outputs: `{"private-marker":`, wantError: "CODER_TEMPLATE_OUTPUTS must be a JSON object"},
		{name: "TrailingJSON", outputs: `{} {"private-marker":1}`, wantError: "CODER_TEMPLATE_OUTPUTS must be a JSON object"},
		{name: "Null", outputs: `null`, wantError: "CODER_TEMPLATE_OUTPUTS must be a JSON object"},
		{name: "Array", outputs: `[]`, wantError: "CODER_TEMPLATE_OUTPUTS must be a JSON object"},
		{name: "String", outputs: `"private-marker"`, wantError: "CODER_TEMPLATE_OUTPUTS must be a JSON object"},
		{name: "Number", outputs: `123`, wantError: "CODER_TEMPLATE_OUTPUTS must be a JSON object"},
		{name: "Boolean", outputs: `true`, wantError: "CODER_TEMPLATE_OUTPUTS must be a JSON object"},
		{name: "Whitespace", outputs: " \n", wantError: "CODER_TEMPLATE_OUTPUTS must be a JSON object"},
		{name: "InvalidRevision", outputs: "{}", revisionID: "private-marker", wantError: "CODER_TEMPLATE_RESOURCE_REVISION_ID must be a UUID"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("CODER_WORKSPACE_TEMPLATE_ID", "")
			t.Setenv("CODER_TEMPLATE_OUTPUTS", tc.outputs)
			t.Setenv("CODER_TEMPLATE_RESOURCE_REVISION_ID", tc.revisionID)
			resp := readTemplateOutputs(t, server, sch)
			if tc.wantError != "" {
				require.Len(t, resp.Diagnostics, 1)
				require.Equal(t, tfprotov5.DiagnosticSeverityError, resp.Diagnostics[0].Severity)
				require.Equal(t, tc.wantError, resp.Diagnostics[0].Summary)
				require.Empty(t, resp.Diagnostics[0].Detail)
				require.NotContains(t, resp.Diagnostics[0].Summary, "private-marker")
				return
			}
			values := templateOutputValues(t, resp, sch)
			require.True(t, values["id"].Equal(tftypes.NewValue(tftypes.String, "import")))
			require.True(t, values["revision_id"].Equal(tftypes.NewValue(tftypes.String, "")))
			require.True(t, values["outputs"].Equal(tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{}}, map[string]tftypes.Value{})))
		})
	}

	t.Run("RefreshedSnapshots", func(t *testing.T) {
		t.Setenv("CODER_WORKSPACE_TEMPLATE_ID", "11111111-1111-4111-8111-111111111111")
		t.Setenv("CODER_TEMPLATE_RESOURCE_REVISION_ID", "22222222-2222-4222-8222-222222222222")
		t.Setenv("CODER_TEMPLATE_OUTPUTS", `{"namespace":"first"}`)
		first := templateOutputValues(t, readTemplateOutputs(t, server, sch), sch)
		secondRevision := "33333333-3333-4333-8333-333333333333"
		t.Setenv("CODER_TEMPLATE_RESOURCE_REVISION_ID", secondRevision)
		// A subsequent build can also change output types and keys.
		t.Setenv("CODER_TEMPLATE_OUTPUTS", `{"namespace":"second","new":true}`)
		second := templateOutputValues(t, readTemplateOutputs(t, server, sch), sch)
		require.True(t, first["id"].Equal(second["id"]))
		require.True(t, second["revision_id"].Equal(tftypes.NewValue(tftypes.String, secondRevision)))
		var outputs map[string]tftypes.Value
		require.NoError(t, second["outputs"].As(&outputs))
		require.True(t, outputs["namespace"].Equal(tftypes.NewValue(tftypes.String, "second")))
		require.True(t, outputs["new"].Equal(tftypes.NewValue(tftypes.Bool, true)))
		third := templateOutputValues(t, readTemplateOutputs(t, server, sch), sch)
		require.True(t, second["outputs"].Equal(third["outputs"]))
	})

	t.Run("UnsetEnvironment", func(t *testing.T) {
		for _, key := range []string{"CODER_WORKSPACE_TEMPLATE_ID", "CODER_TEMPLATE_OUTPUTS", "CODER_TEMPLATE_RESOURCE_REVISION_ID"} {
			t.Setenv(key, "")
			require.NoError(t, os.Unsetenv(key))
		}
		values := templateOutputValues(t, readTemplateOutputs(t, server, sch), sch)
		require.True(t, values["id"].Equal(tftypes.NewValue(tftypes.String, "import")))
		require.True(t, values["revision_id"].Equal(tftypes.NewValue(tftypes.String, "")))
		var outputs map[string]tftypes.Value
		require.NoError(t, values["outputs"].As(&outputs))
		require.Empty(t, outputs)
	})
}

func readTemplateOutputs(t *testing.T, server tfprotov5.ProviderServer, sch *tfprotov5.Schema) *tfprotov5.ReadDataSourceResponse {
	t.Helper()
	config, err := tfprotov5.NewDynamicValue(sch.ValueType(), tftypes.NewValue(sch.ValueType(), map[string]tftypes.Value{
		"id":          tftypes.NewValue(tftypes.String, nil),
		"revision_id": tftypes.NewValue(tftypes.String, nil),
		"outputs":     tftypes.NewValue(tftypes.DynamicPseudoType, nil),
	}))
	require.NoError(t, err)
	resp, err := server.ReadDataSource(t.Context(), &tfprotov5.ReadDataSourceRequest{TypeName: "coder_template_outputs", Config: &config})
	require.NoError(t, err)
	return resp
}

func templateOutputValues(t *testing.T, resp *tfprotov5.ReadDataSourceResponse, sch *tfprotov5.Schema) map[string]tftypes.Value {
	t.Helper()
	require.Empty(t, resp.Diagnostics)
	value, err := resp.State.Unmarshal(sch.ValueType())
	require.NoError(t, err)
	var values map[string]tftypes.Value
	require.NoError(t, value.As(&values))
	return values
}
