package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/mapstructure"
)

type WorkspacePreset struct {
	Name       string            `mapstructure:"name"`
	Parameters map[string]string `mapstructure:"parameters"`
}

func workspacePresetDataSource() *schema.Resource {
	return &schema.Resource{
		SchemaVersion: 1,

		Description: "Use this data source to predefine common configurations for workspaces.",
		ReadContext: func(ctx context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
			var preset WorkspacePreset
			err := mapstructure.Decode(struct {
				Name       interface{}
				Parameters interface{}
			}{
				Name:       rd.Get("name"),
				Parameters: rd.Get("parameters"),
			}, &preset)
			if err != nil {
				return diag.Errorf("decode workspace preset: %s", err)
			}

			if preset.Name == "" {
				return diag.Errorf("workspace preset name must be set")
			}

			if len(preset.Parameters) == 0 {
				return diag.Errorf("workspace preset must define a value for at least one parameter")
			}

			rd.SetId(preset.Name)

			return nil
		},
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Description: "ID of the workspace preset.",
				Computed:    true,
			},
			"name": {
				Type:        schema.TypeString,
				Description: "Name of the workspace preset.",
				Required:    true,
			},
			"parameters": {
				Type:        schema.TypeMap,
				Description: "Parameters of the workspace preset.",
				Required:    true,
			},
		},
	}
}
