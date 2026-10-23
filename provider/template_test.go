package provider_test

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"

	"github.com/coder/terraform-provider-coder/v2/provider"
)

func TestTemplateDataSource(t *testing.T) {
	// Environment-backed reads cannot run in parallel with t.Setenv.
	for _, tc := range []struct {
		name         string
		templateID   string
		templateName string
		wantID       string
		wantName     string
	}{
		{
			name:         "TemplateIdentity",
			templateID:   "11111111-1111-4111-8111-111111111111",
			templateName: "shared-development",
			wantID:       "11111111-1111-4111-8111-111111111111",
			wantName:     "shared-development",
		},
		{name: "Import", wantID: "import", wantName: "default"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("CODER_WORKSPACE_TEMPLATE_ID", tc.templateID)
			t.Setenv("CODER_WORKSPACE_TEMPLATE_NAME", tc.templateName)
			r := provider.New().DataSourcesMap["coder_template"]
			rd := schema.TestResourceDataRaw(t, r.Schema, nil)
			for range 2 {
				require.Empty(t, r.ReadContext(t.Context(), rd, nil))
				require.Equal(t, tc.wantID, rd.Id())
				require.Equal(t, tc.wantName, rd.Get("name"))
			}
		})
	}
}

func TestTemplateDataSourceUnsetEnvironment(t *testing.T) {
	for _, key := range []string{"CODER_WORKSPACE_TEMPLATE_ID", "CODER_WORKSPACE_TEMPLATE_NAME"} {
		t.Setenv(key, "")
		require.NoError(t, os.Unsetenv(key))
	}
	r := provider.New().DataSourcesMap["coder_template"]
	rd := schema.TestResourceDataRaw(t, r.Schema, nil)
	require.Empty(t, r.ReadContext(t.Context(), rd, nil))
	require.Equal(t, "import", rd.Id())
	require.Equal(t, "default", rd.Get("name"))
}
