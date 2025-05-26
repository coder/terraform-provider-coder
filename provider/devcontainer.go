package provider

import (
	"context"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func devcontainerResource() *schema.Resource {
	return &schema.Resource{
		SchemaVersion: 1,

		Description: "Define a Dev Container the agent should know of and attempt to autostart.\n\n-> This resource is only available in Coder v2.21 and later.",
		CreateContext: func(_ context.Context, rd *schema.ResourceData, _ interface{}) diag.Diagnostics {
			rd.SetId(uuid.NewString())

			return nil
		},
		ReadContext:   schema.NoopContext,
		DeleteContext: schema.NoopContext,
		Schema: map[string]*schema.Schema{
			"agent_id": {
				Type:        schema.TypeString,
				Description: "The `id` property of a `coder_agent` resource to associate with.",
				ForceNew:    true,
				Required:    true,
			},
			"workspace_folder": {
				Type:         schema.TypeString,
				Description:  "The workspace folder to for the Dev Container.",
				ForceNew:     true,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"config_path": {
				Type:        schema.TypeString,
				Description: "The path to the Dev Container configuration file (devcontainer.json).",
				ForceNew:    true,
				Optional:    true,
			},
		},
	}
}
