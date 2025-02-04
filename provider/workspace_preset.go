package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
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

			// MinItems doesn't work with maps, so we need to check the length
			// of the map manually. All other validation is handled by the
			// schema.
			if len(preset.Parameters) == 0 {
				return diag.Errorf("expected \"parameters\" to not be an empty map")
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
				Type:         schema.TypeString,
				Description:  "Name of the workspace preset.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"parameters": {
				Type:        schema.TypeMap,
				Description: "Parameters of the workspace preset.",
				Required:    true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					Required:     true,
					ValidateFunc: validation.StringIsNotEmpty,
				},
			},
		},
	}
}
