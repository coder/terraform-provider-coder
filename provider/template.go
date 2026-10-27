package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/coder/terraform-provider-coder/v2/provider/helpers"
)

const templateImportID = "import"

func templateDataSource() *schema.Resource {
	return &schema.Resource{
		SchemaVersion: 1,
		Description:   "Proposed POC data source for template identity in both provisioning scopes: Shared resources (once per template) and Per-workspace resources (each workspace).",
		ReadContext: func(_ context.Context, rd *schema.ResourceData, _ interface{}) diag.Diagnostics {
			if err := rd.Set("name", helpers.OptionalEnvOrDefault("CODER_WORKSPACE_TEMPLATE_NAME", "default")); err != nil {
				return diag.FromErr(err)
			}
			rd.SetId(helpers.OptionalEnvOrDefault("CODER_WORKSPACE_TEMPLATE_ID", templateImportID))
			return nil
		},
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ID of the template, or `import` when unavailable during template import.",
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the template, or `default` when unavailable during template import.",
			},
		},
	}
}
