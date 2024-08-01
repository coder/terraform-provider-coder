package provider

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func workspaceOwnerDataSource() *schema.Resource {
	return &schema.Resource{
		Description: "Use this data source to fetch information about the workspace owner.",
		ReadContext: func(ctx context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
			if idStr := os.Getenv("CODER_WORKSPACE_OWNER_ID"); idStr != "" {
				rd.SetId(idStr)
			} else {
				rd.SetId(uuid.NewString())
			}

			if username := os.Getenv("CODER_WORKSPACE_OWNER"); username != "" {
				_ = rd.Set("name", username)
			} else {
				_ = rd.Set("name", "default")
			}

			if fullname := os.Getenv("CODER_WORKSPACE_OWNER_NAME"); fullname != "" {
				_ = rd.Set("full_name", fullname)
			} else { // compat: field can be blank, fill in default
				_ = rd.Set("full_name", "default")
			}

			if email := os.Getenv("CODER_WORKSPACE_OWNER_EMAIL"); email != "" {
				_ = rd.Set("email", email)
			} else {
				_ = rd.Set("email", "default@example.com")
			}

			_ = rd.Set("ssh_public_key", os.Getenv("CODER_WORKSPACE_OWNER_SSH_PUBLIC_KEY"))
			_ = rd.Set("ssh_private_key", os.Getenv("CODER_WORKSPACE_OWNER_SSH_PRIVATE_KEY"))

			var groups []string
			if groupsRaw, ok := os.LookupEnv("CODER_WORKSPACE_OWNER_GROUPS"); ok {
				if err := json.NewDecoder(strings.NewReader(groupsRaw)).Decode(&groups); err != nil {
					return diag.Errorf("invalid user groups: %s", err.Error())
				}
			}
			_ = rd.Set("groups", groups)

			_ = rd.Set("session_token", os.Getenv("CODER_WORKSPACE_OWNER_SESSION_TOKEN"))
			_ = rd.Set("oidc_access_token", os.Getenv("CODER_WORKSPACE_OWNER_OIDC_ACCESS_TOKEN"))
			_ = rd.Set("oidc_refresh_token", os.Getenv("CODER_WORKSPACE_OWNER_OIDC_REFRESH_TOKEN"))

			return nil
		},
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The UUID of the workspace owner.",
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The username of the user.",
			},
			"full_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The full name of the user.",
			},
			"email": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The email address of the user.",
			},
			"ssh_public_key": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The user's generated SSH public key.",
			},
			"ssh_private_key": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The user's generated SSH private key.",
				Sensitive:   true,
			},
			"groups": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "The groups of which the user is a member.",
			},
			"session_token": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Session token for authenticating with a Coder deployment. It is regenerated every time a workspace is started.",
			},
			"oidc_access_token": {
				Type:     schema.TypeString,
				Computed: true,
				Description: "A valid OpenID Connect access token of the workspace owner. " +
					"This is only available if the workspace owner authenticated with OpenID Connect. " +
					"If a valid token cannot be obtained, this value will be an empty string.",
			},
			"oidc_refresh_token": {
				Type:     schema.TypeString,
				Computed: true,
				Description: "A valid OpenID Connect refresh token of the workspace owner. Can be used to refresh access token if expired " +
					"This is only available if the workspace owner authenticated with OpenID Connect. " +
					"If a valid refresh token cannot be obtained, this value will be an empty string.",
			},
		},
	}
}
