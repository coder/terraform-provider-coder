package provider

import (
	"context"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func envResource() *schema.Resource {
	return &schema.Resource{
		Description: "Use this resource to set an environment variable in a workspace.",
		CreateContext: func(_ context.Context, rd *schema.ResourceData, _ interface{}) diag.Diagnostics {
			rd.SetId(uuid.NewString())

			return nil
		},
		ReadContext:   schema.NoopContext,
		DeleteContext: schema.NoopContext,
		Schema: map[string]*schema.Schema{
			"agent_id": {
				Type:        schema.TypeString,
				Description: `The "id" property of a "coder_agent" resource to associate with.`,
				ForceNew:    true,
				Required:    true,
			},
			"name": {
				Type:        schema.TypeString,
				Description: "The name of the environment variable.",
				ForceNew:    true,
				Required:    true,
			},
			"value": {
				Type:        schema.TypeString,
				Description: "The value of the environment variable.",
				ForceNew:    true,
				Optional:    true,
			},
		},
	}
}
