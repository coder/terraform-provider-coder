package provider

import (
	"context"
	"regexp"

	"github.com/google/uuid"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/coder/terraform-provider-coder/v2/provider/helpers"
)

var (
	// appSlugRegex is the regex used to validate the slug of a coder_app
	// resource. It must be a valid hostname and cannot contain two consecutive
	// hyphens or start/end with a hyphen.
	//
	// This regex is duplicated in the Coder source code, so make sure to update
	// it there as well.
	//
	// There are test cases for this regex in the Coder product.
	appSlugRegex = regexp.MustCompile(`^[a-z0-9](-?[a-z0-9])*$`)
)

const (
	appDisplayNameMaxLength = 64 // database column limit
	appGroupNameMaxLength   = 64
	appTooltipMaxLength     = 2048
)

func appResource() *schema.Resource {
	return &schema.Resource{
		SchemaVersion: 1,

		Description: "Use this resource to define shortcuts to access applications in a workspace.",
		CreateContext: func(c context.Context, resourceData *schema.ResourceData, i any) diag.Diagnostics {
			resourceData.SetId(uuid.NewString())

			diags := diag.Diagnostics{}

			hiddenData := resourceData.Get("hidden")
			if hidden, ok := hiddenData.(bool); !ok {
				return diag.Errorf("hidden should be a bool")
			} else if hidden {
				if _, ok := resourceData.GetOk("display_name"); ok {
					diags = append(diags, diag.Diagnostic{
						Severity: diag.Warning,
						Summary:  "`display_name` set when app is hidden",
					})
				}

				if _, ok := resourceData.GetOk("icon"); ok {
					diags = append(diags, diag.Diagnostic{
						Severity: diag.Warning,
						Summary:  "`icon` set when app is hidden",
					})
				}

				if _, ok := resourceData.GetOk("order"); ok {
					diags = append(diags, diag.Diagnostic{
						Severity: diag.Warning,
						Summary:  "`order` set when app is hidden",
					})
				}
			}

			return diags
		},
		ReadContext: func(c context.Context, resourceData *schema.ResourceData, i any) diag.Diagnostics {
			return nil
		},
		DeleteContext: func(ctx context.Context, rd *schema.ResourceData, i any) diag.Diagnostics {
			return nil
		},
		Schema: map[string]*schema.Schema{
			"agent_id": {
				Type:        schema.TypeString,
				Description: "The `id` property of a `coder_agent` resource to associate with.",
				ForceNew:    true,
				Required:    true,
			},
			"command": {
				Type: schema.TypeString,
				Description: "A command to run in a terminal opening this app. In the web, " +
					"this will open in a new tab. In the CLI, this will SSH and execute the command. " +
					"Either `command` or `url` may be specified, but not both.",
				ConflictsWith: []string{"url"},
				Optional:      true,
				ForceNew:      true,
			},
			"icon": {
				Type: schema.TypeString,
				Description: "A URL to an icon that will display in the dashboard. View built-in " +
					"icons [here](https://github.com/coder/coder/tree/main/site/static/icon). Use a " +
					"built-in icon with `\"${data.coder_workspace.me.access_url}/icon/<path>\"`.",
				ForceNew:     true,
				Optional:     true,
				ValidateFunc: helpers.ValidateURL,
			},
			"slug": {
				Type: schema.TypeString,
				Description: "A hostname-friendly name for the app. This is " +
					"used in URLs to access the app. May contain " +
					"alphanumerics and hyphens. Cannot start/end with a " +
					"hyphen or contain two consecutive hyphens.",
				ForceNew: true,
				Required: true,
				ValidateDiagFunc: func(val any, c cty.Path) diag.Diagnostics {
					valStr, ok := val.(string)
					if !ok {
						return diag.Errorf("expected string, got %T", val)
					}

					if !appSlugRegex.MatchString(valStr) {
						return diag.Errorf(`invalid "coder_app" slug, must be a valid hostname (%q, cannot contain two consecutive hyphens or start/end with a hyphen): %q`, appSlugRegex.String(), valStr)
					}

					return nil
				},
			},
			"display_name": {
				Type:        schema.TypeString,
				Description: "A display name to identify the app. Defaults to the slug.",
				ForceNew:    true,
				Optional:    true,
				ValidateDiagFunc: func(val any, c cty.Path) diag.Diagnostics {
					valStr, ok := val.(string)
					if !ok {
						return diag.Errorf("expected string, got %T", val)
					}

					if len(valStr) > appDisplayNameMaxLength {
						return diag.Errorf("display name is too long (max %d characters)", appDisplayNameMaxLength)
					}
					return nil
				},
			},
			"subdomain": {
				Type: schema.TypeBool,
				Description: "Determines whether the app will be accessed via it's own " +
					"subdomain or whether it will be accessed via a path on Coder. If " +
					"wildcards have not been setup by the administrator then apps with " +
					"`subdomain` set to `true` will not be accessible. Defaults to `false`.",
				ForceNew: true,
				Optional: true,
			},
			"share": {
				Type: schema.TypeString,
				Description: "Determines the level which the application " +
					"is shared at. Valid levels are `\"owner\"` (default), " +
					"`\"authenticated\"` and `\"public\"`. Level `\"owner\"` disables " +
					"sharing on the app, so only the workspace owner can " +
					"access it. Level `\"authenticated\"` shares the app with " +
					"all authenticated users. Level `\"public\"` shares it with " +
					"any user, including unauthenticated users. Permitted " +
					"application sharing levels can be configured site-wide " +
					"via a flag on `coder server` (Enterprise only).",
				ForceNew: true,
				Optional: true,
				Default:  "owner",
				ValidateDiagFunc: func(val any, c cty.Path) diag.Diagnostics {
					valStr, ok := val.(string)
					if !ok {
						return diag.Errorf("expected string, got %T", val)
					}

					switch valStr {
					case "owner", "authenticated", "public":
						return nil
					}

					return diag.Errorf("invalid app share %q, must be one of \"owner\", \"authenticated\", \"public\"", valStr)
				},
			},
			"url": {
				Type: schema.TypeString,
				Description: "An external url if `external=true` or a URL to be proxied to from inside the workspace. " +
					"This should be of the form `http://localhost:PORT[/SUBPATH]`. " +
					"Either `command` or `url` may be specified, but not both.",
				ForceNew:      true,
				Optional:      true,
				ConflictsWith: []string{"command"},
			},
			"external": {
				Type: schema.TypeBool,
				Description: "Specifies whether `url` is opened on the client machine " +
					"instead of proxied through the workspace.",
				Default:       false,
				ForceNew:      true,
				Optional:      true,
				ConflictsWith: []string{"healthcheck", "command", "subdomain", "share"},
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
							Description: "HTTP address used determine the application readiness. A successful health check is a HTTP response code less than 500 returned before `healthcheck.interval` seconds.",
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
			"group": {
				Type:        schema.TypeString,
				Description: "The name of a group that this app belongs to.",
				ForceNew:    true,
				Optional:    true,
				ValidateDiagFunc: func(val any, c cty.Path) diag.Diagnostics {
					valStr, ok := val.(string)
					if !ok {
						return diag.Errorf("expected string, got %T", val)
					}

					if len(valStr) > appGroupNameMaxLength {
						return diag.Errorf("group name is too long (max %d characters)", appGroupNameMaxLength)
					}
					return nil
				},
			},
			"order": {
				Type:        schema.TypeInt,
				Description: "The order determines the position of app in the UI presentation. The lowest order is shown first and apps with equal order are sorted by name (ascending order).",
				ForceNew:    true,
				Optional:    true,
			},
			"hidden": {
				Type:        schema.TypeBool,
				Description: "Determines if the app is visible in the UI (minimum Coder version: v2.16).",
				Default:     false,
				ForceNew:    true,
				Optional:    true,
			},
			"open_in": {
				Type: schema.TypeString,
				Description: "Determines where the app will be opened. Valid values are `\"tab\"` and `\"slim-window\" (default)`. " +
					"`\"tab\"` opens in a new tab in the same browser window. " +
					"`\"slim-window\"` opens a new browser window without navigation controls.",
				ForceNew: true,
				Optional: true,
				Default:  "slim-window",
				ValidateDiagFunc: func(val any, c cty.Path) diag.Diagnostics {
					valStr, ok := val.(string)
					if !ok {
						return diag.Errorf("expected string, got %T", val)
					}

					switch valStr {
					case "tab", "slim-window":
						return nil
					}

					return diag.Errorf(`invalid "coder_app" open_in value, must be one of "tab", "slim-window": %q`, valStr)
				},
			},
			"tooltip": {
				Type:        schema.TypeString,
				Description: "Markdown text that is displayed when hovering over workspace apps.",
				ForceNew:    true,
				Optional:    true,
				ValidateDiagFunc: func(val any, c cty.Path) diag.Diagnostics {
					valStr, ok := val.(string)
					if !ok {
						return diag.Errorf("expected string, got %T", val)
					}

					if len(valStr) > appTooltipMaxLength {
						return diag.Errorf("tooltip is too long (max %d characters)", appTooltipMaxLength)
					}
					return nil
				},
			},
		},
	}
}
