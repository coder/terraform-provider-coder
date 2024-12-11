package provider

import (
	"context"
	"crypto/sha1"
	"encoding/base32"
	"fmt"

	"github.com/coder/terraform-provider-coder/provider/helpers"
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

			pool := resourceData.Get("pool_name").(string)

			// SHA-1 because Base32/64 produces an invalid env infix
			poolName := sha1.Sum([]byte(pool))
			key := fmt.Sprintf("CODER_RESOURCEPOOL_%x_ENTRY_INSTANCE_ID", poolName)
			instanceId, err := helpers.RequireEnv(key)
			if err != nil {
				return diag.Errorf("failed to get instance ID: %s", err.Error())
			}
			decoded, err := base32.StdEncoding.DecodeString(instanceId)
			if err != nil {
				return diag.Errorf("failed to decode instance ID %q: %s", instanceId, err.Error())
			}
			_ = resourceData.Set("instance_id", string(decoded))

			return nil
		},
		ReadContext: func(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
			return nil
		},
		DeleteContext: func(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
			return nil
		},
		Schema: map[string]*schema.Schema{
			"pool_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the pool from which an entry will be claimed.",
			},
			"instance_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The instance ID of the pooled entry resource",
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
				Type:     schema.TypeSet,
				ForceNew: true,
				Optional: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"instance_id": {
							Type:        schema.TypeString,
							Description: "The ID of the instance.",
							ForceNew:    true,
							Required:    true,
						},
					},
				},
			},
		},
	}
}
