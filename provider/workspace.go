package provider

import (
	"context"
	"encoding/json"
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

			ownerGroupsText := os.Getenv("CODER_WORKSPACE_OWNER_GROUPS")
			var ownerGroups []string
			err := json.Unmarshal([]byte(ownerGroupsText), &ownerGroups)
			if err != nil {
				return diag.Errorf("couldn't parse owner groups %q", ownerGroupsText)
			}
			_ = rd.Set("owner_groups", ownerGroups)

			ownerName := os.Getenv("CODER_WORKSPACE_OWNER_NAME")
			_ = rd.Set("owner_name", ownerName)

			ownerID := os.Getenv("CODER_WORKSPACE_OWNER_ID")
			if ownerID == "" {
				ownerID = uuid.Nil.String()
			}
			_ = rd.Set("owner_id", ownerID)

			ownerOIDCAccessToken := os.Getenv("CODER_WORKSPACE_OWNER_OIDC_ACCESS_TOKEN")
			_ = rd.Set("owner_oidc_access_token", ownerOIDCAccessToken)

			name := os.Getenv("CODER_WORKSPACE_NAME")
			if name == "" {
				name = "default"
			}
			rd.Set("name", name)

			sessionToken := os.Getenv("CODER_WORKSPACE_OWNER_SESSION_TOKEN")
			_ = rd.Set("owner_session_token", sessionToken)

			id := os.Getenv("CODER_WORKSPACE_ID")
			if id == "" {
				id = uuid.NewString()
			}
			rd.SetId(id)

			templateID := os.Getenv("CODER_WORKSPACE_TEMPLATE_ID")
			_ = rd.Set("template_id", templateID)

			templateName := os.Getenv("CODER_WORKSPACE_TEMPLATE_NAME")
			_ = rd.Set("template_name", templateName)

			templateVersion := os.Getenv("CODER_WORKSPACE_TEMPLATE_VERSION")
			_ = rd.Set("template_version", templateVersion)

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
			"owner_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the workspace owner.",
			},
			"owner_oidc_access_token": {
				Type:     schema.TypeString,
				Computed: true,
				Description: "A valid OpenID Connect access token of the workspace owner. " +
					"This is only available if the workspace owner authenticated with OpenID Connect. " +
					"If a valid token cannot be obtained, this value will be an empty string.",
			},
			"owner_groups": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "List of groups the workspace owner belongs to.",
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
			"owner_session_token": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Session token for authenticating with a Coder deployment. It is regenerated everytime a workspace is started.",
			},
			"template_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ID of the workspace's template.",
			},
			"template_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the workspace's template.",
			},
			"template_version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Version of the workspace's template.",
			},
		},
	}
}
