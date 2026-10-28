package provider_test

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"

	"github.com/coder/terraform-provider-coder/v2/provider"
)

func TestFoundationDataSource(t *testing.T) {
	// Environment-backed reads cannot run in parallel with t.Setenv.
	const foundationID = "11111111-1111-4111-8111-111111111111"
	const templateID = "22222222-2222-4222-8222-222222222222"
	const snapshotID = "33333333-3333-4333-8333-333333333333"
	for _, tc := range []struct {
		name, id, foundationName, templateIDs, snapshotID, wantID, wantName string
		wantTemplates                                                       []string
		wantError                                                           string
	}{
		{name: "Bound", id: foundationID, foundationName: "shared", templateIDs: `["` + templateID + `"]`, snapshotID: snapshotID, wantID: foundationID, wantName: "shared", wantTemplates: []string{templateID}},
		{name: "Unbound", id: foundationID, foundationName: "shared", templateIDs: `[]`, wantID: foundationID, wantName: "shared"},
		{name: "Import", wantID: "import", wantName: "default"},
		{name: "InvalidID", id: "private-marker", wantError: "Foundation ID in CODER_FOUNDATION_ID must be a UUID"},
		{name: "InvalidSnapshotID", snapshotID: "private-marker", wantError: "Foundation snapshot ID in CODER_FOUNDATION_SNAPSHOT_ID must be a UUID"},
		{name: "MalformedJSON", templateIDs: `["private-marker"`, wantError: "Foundation template IDs in CODER_FOUNDATION_TEMPLATE_IDS must be a JSON array of UUIDs"},
		{name: "Null", templateIDs: `null`, wantError: "Foundation template IDs in CODER_FOUNDATION_TEMPLATE_IDS must be a JSON array of UUIDs"},
		{name: "NullElement", templateIDs: `[null]`, wantError: "Foundation template IDs in CODER_FOUNDATION_TEMPLATE_IDS must be a JSON array of UUIDs"},
		{name: "Object", templateIDs: `{}`, wantError: "Foundation template IDs in CODER_FOUNDATION_TEMPLATE_IDS must be a JSON array of UUIDs"},
		{name: "InvalidTemplateID", templateIDs: `["private-marker"]`, wantError: "Foundation template IDs in CODER_FOUNDATION_TEMPLATE_IDS must be a JSON array of UUIDs"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clearFoundationEnv(t)
			t.Setenv("CODER_FOUNDATION_ID", tc.id)
			t.Setenv("CODER_FOUNDATION_NAME", tc.foundationName)
			t.Setenv("CODER_FOUNDATION_TEMPLATE_IDS", tc.templateIDs)
			t.Setenv("CODER_FOUNDATION_SNAPSHOT_ID", tc.snapshotID)
			t.Setenv("CODER_WORKSPACE_TEMPLATE_ID", templateID)
			t.Setenv("CODER_WORKSPACE_TEMPLATE_NAME", "workspace-template")
			t.Setenv("CODER_SESSION_TOKEN", "private-marker")
			server, sch := foundationServer(t)
			require.Len(t, sch.Block.Attributes, 6)
			for _, attr := range sch.Block.Attributes {
				require.True(t, attr.Computed)
				require.False(t, attr.Optional)
				require.False(t, attr.Required)
			}
			for range 2 {
				resp := readFoundation(t, server, sch)
				if tc.wantError != "" {
					require.Len(t, resp.Diagnostics, 1)
					require.Equal(t, tc.wantError, resp.Diagnostics[0].Summary)
					require.Empty(t, resp.Diagnostics[0].Detail)
					require.NotContains(t, resp.Diagnostics[0].Summary, "private-marker")
					continue
				}
				values := templateOutputValues(t, resp, sch)
				require.True(t, values["id"].Equal(tftypes.NewValue(tftypes.String, tc.wantID)))
				require.True(t, values["name"].Equal(tftypes.NewValue(tftypes.String, tc.wantName)))
				require.True(t, values["snapshot_id"].Equal(tftypes.NewValue(tftypes.String, tc.snapshotID)))
				var templates []tftypes.Value
				require.NoError(t, values["template_ids"].As(&templates))
				require.Len(t, templates, len(tc.wantTemplates))
				for i, id := range tc.wantTemplates {
					require.True(t, templates[i].Equal(tftypes.NewValue(tftypes.String, id)))
				}
				require.True(t, values["outputs"].Equal(tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{}}, map[string]tftypes.Value{})))
			}
		})
	}
}

func TestFoundationDataSourceUnsetEnvironment(t *testing.T) {
	clearFoundationEnv(t)
	server, sch := foundationServer(t)
	values := templateOutputValues(t, readFoundation(t, server, sch), sch)
	require.True(t, values["id"].Equal(tftypes.NewValue(tftypes.String, "import")))
	require.True(t, values["name"].Equal(tftypes.NewValue(tftypes.String, "default")))
}

func clearFoundationEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"CODER_FOUNDATION_ID", "CODER_FOUNDATION_NAME", "CODER_FOUNDATION_TEMPLATE_IDS", "CODER_FOUNDATION_SNAPSHOT_ID", "CODER_TEMPLATE_OUTPUTS", "CODER_FOUNDATION_OUTPUT_METADATA"} {
		t.Setenv(key, "")
		require.NoError(t, os.Unsetenv(key))
	}
}

func foundationServer(t *testing.T) (tfprotov5.ProviderServer, *tfprotov5.Schema) {
	t.Helper()
	server, err := provider.NewMux(t.Context())
	require.NoError(t, err)
	schemas, err := server.GetProviderSchema(t.Context(), &tfprotov5.GetProviderSchemaRequest{})
	require.NoError(t, err)
	require.Empty(t, schemas.Diagnostics)
	sch := schemas.DataSourceSchemas["coder_foundation"]
	require.NotNil(t, sch)
	return server, sch
}

func readFoundation(t *testing.T, server tfprotov5.ProviderServer, sch *tfprotov5.Schema) *tfprotov5.ReadDataSourceResponse {
	t.Helper()
	attributes := make(map[string]tftypes.Value)
	for _, attr := range sch.Block.Attributes {
		attributes[attr.Name] = tftypes.NewValue(attr.Type, nil)
	}
	config, err := tfprotov5.NewDynamicValue(sch.ValueType(), tftypes.NewValue(sch.ValueType(), attributes))
	require.NoError(t, err)
	resp, err := server.ReadDataSource(t.Context(), &tfprotov5.ReadDataSourceRequest{TypeName: "coder_foundation", Config: &config})
	require.NoError(t, err)
	return resp
}
