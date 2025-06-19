package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"golang.org/x/xerrors"

	"github.com/coder/terraform-provider-coder/v2/provider/helpers"
)

func agentResource() *schema.Resource {
	return &schema.Resource{
		SchemaVersion: 1,

		Description: "Use this resource to associate an agent.",
		CreateContext: func(ctx context.Context, resourceData *schema.ResourceData, i interface{}) diag.Diagnostics {
			agentID := uuid.NewString()
			resourceData.SetId(agentID)

			token := agentAuthToken(ctx, "")
			err := resourceData.Set("token", token)
			if err != nil {
				return diag.FromErr(err)
			}

			if _, ok := resourceData.GetOk("display_apps"); !ok {
				err = resourceData.Set("display_apps", []interface{}{
					map[string]bool{
						"vscode":                 true,
						"vscode_insiders":        false,
						"web_terminal":           true,
						"ssh_helper":             true,
						"port_forwarding_helper": true,
					},
				})
				if err != nil {
					return diag.FromErr(err)
				}
			}

			if name, ok := resourceData.GetOk("name"); ok {
				err := resourceData.Set("name", name.(string))
				if err != nil {
					return diag.FromErr(err)
				}
			}

			return updateInitScript(resourceData, i)
		},
		ReadWithoutTimeout: func(ctx context.Context, resourceData *schema.ResourceData, i interface{}) diag.Diagnostics {
			token := agentAuthToken(ctx, "")
			err := resourceData.Set("token", token)
			if err != nil {
				return diag.FromErr(err)
			}

			if _, ok := resourceData.GetOk("display_apps"); !ok {
				err = resourceData.Set("display_apps", []interface{}{
					map[string]bool{
						"vscode":                 true,
						"vscode_insiders":        false,
						"web_terminal":           true,
						"ssh_helper":             true,
						"port_forwarding_helper": true,
					},
				})
				if err != nil {
					return diag.FromErr(err)
				}
			}

			if name, ok := resourceData.GetOk("name"); ok {
				err := resourceData.Set("name", name.(string))
				if err != nil {
					return diag.FromErr(err)
				}
			}

			return updateInitScript(resourceData, i)
		},
		DeleteContext: func(ctx context.Context, resourceData *schema.ResourceData, i interface{}) diag.Diagnostics {
			return nil
		},
		Schema: map[string]*schema.Schema{
			"api_key_scope": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "all",
				ForceNew:    true,
				Description: "Controls what API routes the agent token can access. Options: `all` (full access) or `no_user_data` (blocks `/external-auth`, `/gitsshkey`, and `/gitauth` routes)",
				ValidateFunc: validation.StringInSlice([]string{
					"all",
					"no_user_data",
				}, false),
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The name of the agent.",
			},
			"init_script": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Run this script on startup of an instance to initialize the agent.",
			},
			"arch": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The architecture of the agent.",
				Default:      "amd64",
				ValidateFunc: validation.StringInSlice([]string{"amd64", "arm64"}, false),
			},
			"os": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The operating system of the agent.",
				Default:      "linux",
				ValidateFunc: validation.StringInSlice([]string{"linux", "darwin", "windows"}, false),
			},
			"dir": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The directory to install the agent to.",
			},
			"startup_script": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A script to run on startup of the agent.",
			},
			"display_apps": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"vscode": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Whether to display the VSCode app.",
						},
						"vscode_insiders": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Whether to display the VSCode Insiders app.",
						},
						"web_terminal": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Whether to display the web terminal app.",
						},
						"ssh_helper": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Whether to display the SSH helper app.",
						},
						"port_forwarding_helper": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Whether to display the port forwarding helper app.",
						},
					},
				},
			},
			"token": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "The token to use to authenticate the agent.",
			},
			"metadata": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"startup_script_behavior": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "blocking",
							Description: "The behavior of the startup script. Can be `blocking` or `non-blocking`.",
							ValidateFunc: validation.StringInSlice([]string{
								"blocking",
								"non-blocking",
							}, false),
						},
						"startup_script_timeout": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     300,
							Description: "The timeout in seconds for the startup script.",
						},
					},
				},
			},
			"connection_timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     10,
				Description: "The timeout in seconds for the agent to connect to the Coder deployment.",
			},
			"login_before_ready": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to allow logging in to the agent before it is ready.",
			},
		},
	}
}

func updateInitScript(resourceData *schema.ResourceData, i interface{}) diag.Diagnostics {
	p := i.(*provider)

	arch := resourceData.Get("arch").(string)
	os := resourceData.Get("os").(string)
	dir := resourceData.Get("dir").(string)
	startupScript := resourceData.Get("startup_script").(string)
	displayApps := resourceData.Get("display_apps").([]interface{})
	token := resourceData.Get("token").(string)
	metadata := resourceData.Get("metadata").([]interface{})
	connectionTimeout := resourceData.Get("connection_timeout").(int)
	loginBeforeReady := resourceData.Get("login_before_ready").(bool)

	var displayAppsMap map[string]bool
	if len(displayApps) > 0 {
		displayAppsMap = displayApps[0].(map[string]bool)
	}

	var metadataMap map[string]interface{}
	if len(metadata) > 0 {
		metadataMap = metadata[0].(map[string]interface{})
	}

	initScript, err := helpers.AgentInitScript(
		p.client.URL(),
		token,
		arch,
		os,
		dir,
		startupScript,
		displayAppsMap,
		metadataMap,
		connectionTimeout,
		loginBeforeReady,
	)
	if err != nil {
		return diag.FromErr(err)
	}

	err = resourceData.Set("init_script", initScript)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func agentAuthToken(ctx context.Context, agentID string) string {
	// This is a bit of a hack to get a stable token for the agent.
	// We use the agent ID as the salt for the token, so that the token
	// is the same for the same agent.
	// This is not a security risk, as the token is only used to
	// authenticate the agent to the Coder deployment.
	// The token is not used to authenticate the user to the agent.
	// The user is authenticated to the agent using their Coder session token.
	// The agent token is only used to identify the agent to the Coder
	// deployment.
	// The agent token is not sensitive, as it does not grant any
	// permissions to the user.
	// The agent token is only used to identify the agent to the Coder
	// deployment.
	// The agent token is not sensitive, as it does not grant any
	// permissions to the user.
	h := sha256.New()
	h.Write([]byte(agentID))
	return hex.EncodeToString(h.Sum(nil))
}
