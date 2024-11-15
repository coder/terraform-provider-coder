package provider

import (
	"context"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func poolResourceClaimResource() *schema.Resource {
	return &schema.Resource{
		SchemaVersion: 1,

		Description: "Use this resource to claim a resource pool entry.",
		CreateContext: func(_ context.Context, resourceData *schema.ResourceData, i interface{}) diag.Diagnostics {
			resourceData.SetId(uuid.NewString())
			return nil
		},
		ReadContext: func(ctx context.Context, resourceData *schema.ResourceData, i interface{}) diag.Diagnostics {
			// poolb64 := base64.StdEncoding.EncodeToString([]byte(resourceData.Get("pool_name").(string)))
			// key := fmt.Sprintf("CODER_RESOURCE_POOL_%s_ENTRY_OBJECT_ID", poolb64)
			// objectId, err := helpers.RequireEnv(key)
			// if err != nil {
			// 	return diag.Errorf("resource pool entry object ID not found in env %q: %s", key, err)
			// }
			//
			// if err = resourceData.Set("object_id", objectId); err != nil {
			// 	return diag.Errorf("failed to set resource pool entry object ID: %s", err)
			// }
			return nil
		},
		DeleteContext: func(ctx context.Context, resourceData *schema.ResourceData, i interface{}) diag.Diagnostics {
			return nil
		},
		Schema: map[string]*schema.Schema{
			"pool_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the pool from which an entry will be claimed.",
			},
			"object_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The object ID of the pooled entry resource",
			},
		},
	}
}

func poolResourceClaimableResource() *schema.Resource {
	return &schema.Resource{
		SchemaVersion: 1,

		Description: "Use this resource to specify which resources are claimable by workspaces.",
		CreateContext: func(_ context.Context, resourceData *schema.ResourceData, i interface{}) diag.Diagnostics {
			resourceData.SetId(uuid.NewString())
			return nil
		},
		ReadContext: func(ctx context.Context, resourceData *schema.ResourceData, i interface{}) diag.Diagnostics {
			return nil
		},
		DeleteContext: func(ctx context.Context, resourceData *schema.ResourceData, i interface{}) diag.Diagnostics {
			return nil
		},
		Schema: map[string]*schema.Schema{
			"compute": {
				Type:          schema.TypeSet,
				ForceNew:      true,
				Optional:      true,
				ConflictsWith: []string{"other"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"instance_id": {
							Type:        schema.TypeString,
							Description: "The ID of the compute instance (container/VM/etc).",
							ForceNew:    true,
							Required:    true,
						},
						"agent_id": {
							Type:        schema.TypeString,
							Description: "The ID of the agent running inside the compute instance.",
							ForceNew:    true,
							Required:    true,
						},
					},
				},
			},
			"other": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},
		},
	}
}
