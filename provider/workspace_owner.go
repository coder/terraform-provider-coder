package provider

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func workspaceOwnerDataSource() *schema.Resource {
	return &schema.Resource{
		Description: "Use this data source to fetch information about the workspace owner.",
		ReadContext: func(ctx context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
			diags := diag.Diagnostics{}

			ownerID := os.Getenv("CODER_WORKSPACE_OWNER_ID")
			if ownerID == "" {
				ownerID = uuid.NewString()
			}
			rd.SetId(ownerID)

			username := os.Getenv("CODER_WORKSPACE_OWNER")
			if username == "" {
				username = "default"
			}
			_ = rd.Set("name", username)

			// A delegated identity is an identity created under a user,
			// for example for an agent. Its root user answers for it.
			// Any other owner is their own root user, so templates can
			// read the root user without asking which kind of owner
			// they have. Deployments that predate delegated identities
			// set none of these.
			delegated := false
			if delegatedRaw := os.Getenv("CODER_WORKSPACE_OWNER_DELEGATED"); delegatedRaw != "" {
				var err error
				delegated, err = strconv.ParseBool(delegatedRaw)
				if err != nil {
					return diag.Errorf("invalid delegated owner flag: %s", err.Error())
				}
			}
			_ = rd.Set("delegated", delegated)

			// Only a delegated identity has a parent. The parent is the
			// root user, or another delegated identity in between.
			_ = rd.Set("parent_id", os.Getenv("CODER_WORKSPACE_OWNER_PARENT_ID"))

			// Labels say what a delegated identity is for. Whoever created
			// the identity set them, and Coder does not verify them.
			labels := map[string]string{}
			if labelsRaw := os.Getenv("CODER_WORKSPACE_OWNER_LABELS"); labelsRaw != "" {
				if err := json.NewDecoder(strings.NewReader(labelsRaw)).Decode(&labels); err != nil {
					return diag.Errorf("invalid owner labels: %s", err.Error())
				}
			}
			_ = rd.Set("labels", labels)

			rootUserID := os.Getenv("CODER_WORKSPACE_OWNER_ROOT_USER_ID")
			if rootUserID == "" {
				rootUserID = ownerID
			}
			_ = rd.Set("root_user_id", rootUserID)

			rootUserName := os.Getenv("CODER_WORKSPACE_OWNER_ROOT_USER_NAME")
			if rootUserName == "" {
				rootUserName = username
			}
			_ = rd.Set("root_user_name", rootUserName)

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

			if loginType := os.Getenv("CODER_WORKSPACE_OWNER_LOGIN_TYPE"); loginType != "" {
				_ = rd.Set("login_type", loginType)
			}

			var rbacRoles []map[string]string
			if rolesRaw, ok := os.LookupEnv("CODER_WORKSPACE_OWNER_RBAC_ROLES"); ok {
				if err := json.NewDecoder(strings.NewReader(rolesRaw)).Decode(&rbacRoles); err != nil {
					return diag.Errorf("invalid user rbac roles: %s", err.Error())
				}
			}
			_ = rd.Set("rbac_roles", rbacRoles)

			return diags
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
				Sensitive:   true,
			},
			"oidc_access_token": {
				Type:     schema.TypeString,
				Computed: true,
				Description: "A valid OpenID Connect access token of the workspace owner. " +
					"This is only available if the workspace owner authenticated with OpenID Connect. " +
					"If a valid token cannot be obtained, this value will be an empty string.",
				Sensitive: true,
			},
			"login_type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The type of login the user has.",
			},
			"delegated": {
				Type:     schema.TypeBool,
				Computed: true,
				Description: "Whether the workspace owner is a delegated identity: an identity created under a user that can never do more than that user, such as the identity of an agent. " +
					"A delegated identity has no email address and none of its root user's credentials.",
			},
			"parent_id": {
				Type:     schema.TypeString,
				Computed: true,
				Description: "The UUID of the identity the workspace owner was created under: its root user, or another delegated identity in between. " +
					"This is empty unless the owner is a delegated identity.",
			},
			"labels": {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed: true,
				Description: "What whoever created the delegated identity said it is for, such as the session it serves. " +
					"Coder does not verify labels, so treat them as untrusted input. This is empty unless the owner is a delegated identity.",
			},
			"root_user_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The UUID of the user who answers for the workspace owner. This is the owner, unless the owner is a delegated identity.",
			},
			"root_user_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The username of the user who answers for the workspace owner. This is the owner, unless the owner is a delegated identity.",
			},
			"rbac_roles": {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the RBAC role.",
						},
						"org_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The organization ID associated with the RBAC role.",
						},
					},
				},
				Computed:    true,
				Description: "The RBAC roles of which the user is assigned.",
			},
		},
	}
}
