package provider

import (
	"context"
	"net/url"

	"github.com/google/uuid"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func appResource() *schema.Resource {
	return &schema.Resource{
		Description: "Use this resource to define shortcuts to access applications in a workspace.",
		CreateContext: func(c context.Context, resourceData *schema.ResourceData, i interface{}) diag.Diagnostics {
			resourceData.SetId(uuid.NewString())
			return nil
		},
		ReadContext: func(c context.Context, resourceData *schema.ResourceData, i interface{}) diag.Diagnostics {
			return nil
		},
		DeleteContext: func(ctx context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
			return nil
		},
		Schema: map[string]*schema.Schema{
			"agent_id": {
				Type:        schema.TypeString,
				Description: `The "id" property of a "coder_agent" resource to associate with.`,
				ForceNew:    true,
				Required:    true,
			},
			"command": {
				Type: schema.TypeString,
				Description: "A command to run in a terminal opening this app. In the web, " +
					"this will open in a new tab. In the CLI, this will SSH and execute the command. " +
					"Either \"command\" or \"url\" may be specified, but not both.",
				ConflictsWith: []string{"url"},
				Optional:      true,
				ForceNew:      true,
			},
			"icon": {
				Type: schema.TypeString,
				Description: "A URL to an icon that will display in the dashboard. View built-in " +
					"icons here: https://github.com/coder/coder/tree/main/site/static/icons. Use a " +
					"built-in icon with `data.coder_workspace.me.access_url + \"/icons/<path>\"`.",
				ForceNew: true,
				Optional: true,
				ValidateFunc: func(i interface{}, s string) ([]string, []error) {
					_, err := url.Parse(s)
					if err != nil {
						return nil, []error{err}
					}
					return nil, nil
				},
			},
			"name": {
				Type:        schema.TypeString,
				Description: "A display name to identify the app.",
				ForceNew:    true,
				Optional:    true,
			},
			"relative_path": {
				Type:       schema.TypeBool,
				Deprecated: "`relative_path` on apps is deprecated, use `subdomain` instead.",
				Description: "Specifies whether the URL will be accessed via a relative " +
					"path or wildcard. Use if wildcard routing is unavailable. Defaults to true.",
				ForceNew: true,
				Optional: true,
			},
			"subdomain": {
				Type: schema.TypeBool,
				Description: "Determines whether the app will be accessed via it's own " +
					"subdomain or whether it will be accessed via a path on Coder. If " +
					"wildcards have not been setup by the administrator then apps with " +
					"\"subdomain\" set to true will not be accessible. Defaults to false.",
				ForceNew: true,
				Optional: true,
			},
			"share": {
				Type: schema.TypeString,
				Description: `Determines the "level" which the application ` +
					`is shared at. Valid levels are "owner", "template", ` +
					`"authenticated" and "public". Level "owner" disables ` +
					"sharing on the app, so only the workspace owner can " +
					`access it. Level "template" shares the app with all users ` +
					`that can read the workspace's template. Level ` +
					`"authenticated" shares the app with all authenticated ` +
					`users. Level "public" shares it with any user, ` +
					"including unauthenticated users. Permitted application " +
					"sharing levels can be configured site-wide via a flag " +
					`on "coder server" (enterprise only). Defaults to ` +
					`"owner" (sharing disabled).`,
				ForceNew: true,
				Optional: true,
				Default:  "owner",
				ValidateDiagFunc: func(val interface{}, c cty.Path) diag.Diagnostics {
					valStr, ok := val.(string)
					if !ok {
						return diag.Errorf("expected string, got %T", val)
					}

					switch valStr {
					case "owner", "template", "authenticated", "public":
						return nil
					}

					return diag.Errorf(`invalid app share %q, must be one of "owner", "template", "authenticated", "public"`, valStr)
				},
			},
			"url": {
				Type: schema.TypeString,
				Description: "A URL to be proxied to from inside the workspace. " +
					"Either \"command\" or \"url\" may be specified, but not both.",
				ForceNew:      true,
				Optional:      true,
				ConflictsWith: []string{"command"},
			},
			"healthcheck": {
				Type:          schema.TypeSet,
				Description:   "HTTP health checking to determine the application readiness.",
				ForceNew:      true,
				Optional:      true,
				MaxItems:      1,
				ConflictsWith: []string{"command"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"url": {
							Type:        schema.TypeString,
							Description: "HTTP address used determine the application readiness. A successful health check is a HTTP response code less than 500 returned before healthcheck.interval seconds.",
							ForceNew:    true,
							Required:    true,
						},
						"interval": {
							Type:        schema.TypeInt,
							Description: "Duration in seconds to wait between healthcheck requests.",
							ForceNew:    true,
							Required:    true,
						},
						"threshold": {
							Type:        schema.TypeInt,
							Description: "Number of consecutive heathcheck failures before returning an unhealthy status.",
							ForceNew:    true,
							Required:    true,
						},
					},
				},
			},
		},
	}
}
