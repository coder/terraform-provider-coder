package provider

import (
	"context"
	"os"
	"reflect"
	"strconv"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func workspaceDataSource() *schema.Resource {
	return &schema.Resource{
		Description: "Use this data source to get information for the active workspace build.",
		ReadContext: func(c context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
			transition := os.Getenv("CODER_WORKSPACE_TRANSITION")
			if transition == "" {
				// Default to start!
				transition = "start"
			}
			_ = rd.Set("transition", transition)
			count := 0
			if transition == "start" {
				count = 1
			}
			_ = rd.Set("start_count", count)

			owner := os.Getenv("CODER_WORKSPACE_OWNER")
			if owner == "" {
				owner = "default"
			}
			_ = rd.Set("owner", owner)

			ownerEmail := os.Getenv("CODER_WORKSPACE_OWNER_EMAIL")
			_ = rd.Set("owner_email", ownerEmail)

			ownerID := os.Getenv("CODER_WORKSPACE_OWNER_ID")
			if ownerID == "" {
				ownerID = uuid.Nil.String()
			}
			_ = rd.Set("owner_id", ownerID)

			name := os.Getenv("CODER_WORKSPACE_NAME")
			if name == "" {
				name = "default"
			}
			rd.Set("name", name)

			id := os.Getenv("CODER_WORKSPACE_ID")
			if id == "" {
				id = uuid.NewString()
			}
			rd.SetId(id)

			config, valid := i.(config)
			if !valid {
				return diag.Errorf("config was unexpected type %q", reflect.TypeOf(i).String())
			}
			rd.Set("access_url", config.URL.String())

			rawPort := config.URL.Port()
			if rawPort == "" {
				rawPort = "80"
				if config.URL.Scheme == "https" {
					rawPort = "443"
				}
			}
			port, err := strconv.Atoi(rawPort)
			if err != nil {
				return diag.Errorf("couldn't parse port %q", port)
			}
			rd.Set("access_port", port)

			return nil
		},
		Schema: map[string]*schema.Schema{
			"access_url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The access URL of the Coder deployment provisioning this workspace.",
			},
			"access_port": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The access port of the Coder deployment provisioning this workspace.",
			},
			"start_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: `A computed count based on "transition" state. If "start", count will equal 1.`,
			},
			"transition": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: `Either "start" or "stop". Use this to start/stop resources with "count".`,
			},
			"owner": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Username of the workspace owner.",
			},
			"owner_email": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Email address of the workspace owner.",
			},
			"owner_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "UUID of the workspace owner.",
			},
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "UUID of the workspace.",
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the workspace.",
			},
		},
	}
}
