package provider

import (
	"context"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func fileResource() *schema.Resource {
	return &schema.Resource{
		Description: "Use this resource to place a file inside of a workspace.",
		CreateContext: func(ctx context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
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
			"path": {
				Type:        schema.TypeString,
				Description: "The path of the file to place in the workspace.",
				ForceNew:    true,
				Required:    true,
			},
			"content": {
				Type:        schema.TypeString,
				Description: "The content of the file to place in the workspace.",
				ForceNew:    true,
				Required:    true,
			},
			"mode": {
				Type:         schema.TypeInt,
				Description:  "The mode of the file to place in the workspace.",
				ForceNew:     true,
				Optional:     true,
				ValidateFunc: validation.IntBetween(0, 0777),
			},
		},
	}
}
