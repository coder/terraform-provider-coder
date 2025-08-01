package provider

import (
	"context"

	"github.com/coder/terraform-provider-coder/v2/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func secretDataSource() *schema.Resource {
	return &schema.Resource{
		SchemaVersion: 1,

		Description: "",
		ReadContext: func(ctx context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
			envName, ok := rd.Get("env_name").(string)
			if !ok || envName == "" {
				return diag.Errorf("env_name is required")
			}
			envVal := helpers.OptionalEnv(envName)

			rd.Set("value", envVal)
			rd.SetId(envName)
			return nil
		},
		Schema: map[string]*schema.Schema{
			"env_name": {
				Type:        schema.TypeString,
				Description: "",
				Required:    true,
			},
			"value": {
				Type:        schema.TypeString,
				Description: "",
				Computed:    true,
			},
		},
	}
}
