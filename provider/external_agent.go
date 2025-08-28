package provider

import (
	"context"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func externalAgentResource() *schema.Resource {
	return &schema.Resource{
		SchemaVersion: 1,

		Description: "Define an external agent to be used in a workspace.\n\n~> **Warning:** External agents require a [Premium](https://coder.com/pricing) Coder license.",
		CreateContext: func(ctx context.Context, rd *schema.ResourceData, _ interface{}) diag.Diagnostics {
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
		},
	}
}
