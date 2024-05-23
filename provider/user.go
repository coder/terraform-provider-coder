package provider

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type Role struct {
	Name        string `json:"name"`
	DisplayName string `json:"display-name"`
}

func userDataSource() *schema.Resource {
	return &schema.Resource{
		Description: "Use this data source to fetch information about a user.",
		ReadContext: func(ctx context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
			if idStr, ok := os.LookupEnv("CODER_USER_ID"); !ok {
				return diag.Errorf("missing user id")
			} else {
				rd.SetId(idStr)
			}

			if username, ok := os.LookupEnv("CODER_USER_NAME"); !ok {
				return diag.Errorf("missing user username")
			} else {
				_ = rd.Set("name", username)
			}

			if fullname, ok := os.LookupEnv("CODER_USER_FULL_NAME"); !ok {
				_ = rd.Set("name", "default") // compat
			} else {
				_ = rd.Set("full_name", fullname)
			}

			if email, ok := os.LookupEnv("CODER_USER_EMAIL"); !ok {
				return diag.Errorf("missing user email")
			} else {
				_ = rd.Set("email", email)
			}

			if sshPubKey, ok := os.LookupEnv("CODER_USER_SSH_PUBLIC_KEY"); !ok {
				return diag.Errorf("missing user ssh_public_key")
			} else {
				_ = rd.Set("ssh_public_key", sshPubKey)
			}

			if sshPrivKey, ok := os.LookupEnv("CODER_USER_SSH_PRIVATE_KEY"); !ok {
				return diag.Errorf("missing user ssh_private_key")
			} else {
				_ = rd.Set("ssh_private_key", sshPrivKey)
			}

			groupsRaw, ok := os.LookupEnv("CODER_USER_GROUPS")
			if !ok {
				return diag.Errorf("missing user groups")
			}
			var groups []string
			if err := json.NewDecoder(strings.NewReader(groupsRaw)).Decode(&groups); err != nil {
				return diag.Errorf("invalid user groups: %s", err.Error())
			} else {
				_ = rd.Set("groups", groups)
			}

			return nil
		},
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The UUID of the user.",
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
		},
	}
}
