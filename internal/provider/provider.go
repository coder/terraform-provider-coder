package provider

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

type config struct {
	URL *url.URL
}

// New returns a new Terraform provider.
func New() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"url": {
				Type:        schema.TypeString,
				Description: "The URL to access Coder.",
				Optional:    true,
				// The "CODER_AGENT_URL" environment variable is used by default
				// as the Access URL when generating scripts.
				DefaultFunc: schema.EnvDefaultFunc("CODER_AGENT_URL", "https://mydeployment.coder.com"),
				ValidateFunc: func(i interface{}, s string) ([]string, []error) {
					_, err := url.Parse(s)
					if err != nil {
						return nil, []error{err}
					}
					return nil, nil
				},
			},
		},
		ConfigureContextFunc: func(c context.Context, resourceData *schema.ResourceData) (interface{}, diag.Diagnostics) {
			rawURL, ok := resourceData.Get("url").(string)
			if !ok {
				return nil, diag.Errorf("unexpected type %q for url", reflect.TypeOf(resourceData.Get("url")).String())
			}
			if rawURL == "" {
				return nil, diag.Errorf("CODER_AGENT_URL must not be empty; got %q", rawURL)
			}
			parsed, err := url.Parse(resourceData.Get("url").(string))
			if err != nil {
				return nil, diag.FromErr(err)
			}
			return config{
				URL: parsed,
			}, nil
		},
		DataSourcesMap: map[string]*schema.Resource{
			"coder_workspace": {
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

					return nil
				},
				Schema: map[string]*schema.Schema{
					"access_url": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "The access URL of the Coder deployment provisioning this workspace.",
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
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"coder_agent": {
				Description: "Use this resource to associate an agent.",
				CreateContext: func(c context.Context, resourceData *schema.ResourceData, i interface{}) diag.Diagnostics {
					// This should be a real authentication token!
					resourceData.SetId(uuid.NewString())
					err := resourceData.Set("token", uuid.NewString())
					if err != nil {
						return diag.FromErr(err)
					}
					return updateInitScript(resourceData, i)
				},
				ReadContext: func(c context.Context, resourceData *schema.ResourceData, i interface{}) diag.Diagnostics {
					return updateInitScript(resourceData, i)
				},
				DeleteContext: func(c context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
					return nil
				},
				Schema: map[string]*schema.Schema{
					"init_script": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "Run this script on startup of an instance to initialize the agent.",
					},
					"arch": {
						Type:         schema.TypeString,
						ForceNew:     true,
						Required:     true,
						Description:  `The architecture the agent will run on. Must be one of: "amd64", "armv7", "arm64".`,
						ValidateFunc: validation.StringInSlice([]string{"amd64", "armv7", "arm64"}, false),
					},
					"auth": {
						Type:         schema.TypeString,
						Default:      "token",
						ForceNew:     true,
						Optional:     true,
						Description:  `The authentication type the agent will use. Must be one of: "token", "google-instance-identity", "aws-instance-identity", "azure-instance-identity".`,
						ValidateFunc: validation.StringInSlice([]string{"token", "google-instance-identity", "aws-instance-identity", "azure-instance-identity"}, false),
					},
					"dir": {
						Type:        schema.TypeString,
						ForceNew:    true,
						Optional:    true,
						Description: "The starting directory when a user creates a shell session. Defaults to $HOME.",
					},
					"env": {
						ForceNew:    true,
						Description: "A mapping of environment variables to set inside the workspace.",
						Type:        schema.TypeMap,
						Optional:    true,
					},
					"os": {
						Type:         schema.TypeString,
						ForceNew:     true,
						Required:     true,
						Description:  `The operating system the agent will run on. Must be one of: "linux", "darwin", or "windows".`,
						ValidateFunc: validation.StringInSlice([]string{"linux", "darwin", "windows"}, false),
					},
					"startup_script": {
						ForceNew:    true,
						Description: "A script to run after the agent starts.",
						Type:        schema.TypeString,
						Optional:    true,
					},
					"token": {
						ForceNew:    true,
						Description: `Set the environment variable "CODER_AGENT_TOKEN" with this token to authenticate an agent.`,
						Type:        schema.TypeString,
						Computed:    true,
					},
				},
			},
			"coder_agent_instance": {
				Description: "Use this resource to associate an instance ID with an agent for zero-trust " +
					"authentication. This association is done automatically for \"google_compute_instance\", " +
					"\"aws_instance\", \"azurerm_linux_virtual_machine\", and " +
					"\"azurerm_windows_virtual_machine\" resources.",
				CreateContext: func(c context.Context, resourceData *schema.ResourceData, i interface{}) diag.Diagnostics {
					resourceData.SetId(uuid.NewString())
					return nil
				},
				ReadContext: func(c context.Context, resourceData *schema.ResourceData, i interface{}) diag.Diagnostics {
					return nil
				},
				DeleteContext: func(c context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
					return nil
				},
				Schema: map[string]*schema.Schema{
					"agent_id": {
						Type:        schema.TypeString,
						Description: `The "id" property of a "coder_agent" resource to associate with.`,
						ForceNew:    true,
						Required:    true,
					},
					"instance_id": {
						ForceNew:    true,
						Required:    true,
						Description: `The instance identifier of a provisioned resource.`,
						Type:        schema.TypeString,
					},
				},
			},
			"coder_app": {
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
						Type: schema.TypeBool,
						Description: "Specifies whether the URL will be accessed via a relative " +
							"path or wildcard. Use if wildcard routing is unavailable.",
						ForceNew:      true,
						Optional:      true,
						ConflictsWith: []string{"command"},
					},
					"url": {
						Type: schema.TypeString,
						Description: "A URL to be proxied to from inside the workspace. " +
							"Either \"command\" or \"url\" may be specified, but not both.",
						ForceNew:      true,
						Optional:      true,
						ConflictsWith: []string{"command"},
					},
				},
			},
			"coder_metadata": {
				Description: "Use this resource to attach key/value pairs to a resource. They will be " +
					"displayed in the Coder dashboard.",
				CreateContext: func(c context.Context, resourceData *schema.ResourceData, i interface{}) diag.Diagnostics {
					resourceData.SetId(uuid.NewString())

					pairs, err := populateIsNull(resourceData)
					if err != nil {
						return errorAsDiagnostics(err)
					}
					err = resourceData.Set("pair", pairs)
					if err != nil {
						return errorAsDiagnostics(err)
					}

					return nil
				},
				ReadContext: func(c context.Context, resourceData *schema.ResourceData, i interface{}) diag.Diagnostics {
					return nil
				},
				DeleteContext: func(ctx context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
					return nil
				},
				Schema: map[string]*schema.Schema{
					"resource_id": {
						Type:        schema.TypeString,
						Description: "The \"id\" property of another resource that metadata should be attached to.",
						ForceNew:    true,
						Required:    true,
					},
					"pair": {
						Type:        schema.TypeList,
						Description: "Each \"pair\" block defines a single key/value metadata pair.",
						ForceNew:    true,
						Required:    true,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"key": {
									Type:        schema.TypeString,
									Description: "The key of this metadata item.",
									ForceNew:    true,
									Required:    true,
								},
								"value": {
									Type:        schema.TypeString,
									Description: "The value of this metadata item.",
									ForceNew:    true,
									Optional:    true,
								},
								"sensitive": {
									Type: schema.TypeBool,
									Description: "Set to \"true\" to for items such as API keys whose values should be " +
										"hidden from view by default. Note that this does not prevent metadata from " +
										"being retrieved using the API, so it is not suitable for secrets that should " +
										"not be exposed to workspace users.",
									ForceNew: true,
									Optional: true,
									Default:  false,
								},
								"is_null": {
									Type:     schema.TypeBool,
									ForceNew: true,
									Computed: true,
								},
							},
						},
					},
				},
			},
		},
	}
}

// updateInitScript fetches parameters from a "coder_agent" to produce the
// agent script from environment variables.
func updateInitScript(resourceData *schema.ResourceData, i interface{}) diag.Diagnostics {
	config, valid := i.(config)
	if !valid {
		return diag.Errorf("config was unexpected type %q", reflect.TypeOf(i).String())
	}
	auth, valid := resourceData.Get("auth").(string)
	if !valid {
		return diag.Errorf("auth was unexpected type %q", reflect.TypeOf(resourceData.Get("auth")))
	}
	operatingSystem, valid := resourceData.Get("os").(string)
	if !valid {
		return diag.Errorf("os was unexpected type %q", reflect.TypeOf(resourceData.Get("os")))
	}
	arch, valid := resourceData.Get("arch").(string)
	if !valid {
		return diag.Errorf("arch was unexpected type %q", reflect.TypeOf(resourceData.Get("arch")))
	}
	accessURL, err := config.URL.Parse("/")
	if err != nil {
		return diag.Errorf("parse access url: %s", err)
	}
	script := os.Getenv(fmt.Sprintf("CODER_AGENT_SCRIPT_%s_%s", operatingSystem, arch))
	if script != "" {
		script = strings.ReplaceAll(script, "${ACCESS_URL}", accessURL.String())
		script = strings.ReplaceAll(script, "${AUTH_TYPE}", auth)
	}
	err = resourceData.Set("init_script", script)
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}

// populateIsNull reads the raw plan for a coder_metadata resource being created,
// figures out which items have null "value"s, and augments them by setting the
// "is_null" field to true. This ugly hack is necessary because terraform-plugin-sdk
// is designed around a old version of Terraform that didn't support nullable fields,
// and it doesn't correctly propagate null values for primitive types.
// Returns an interface{} representing the new value of the "pair" field, or an error.
func populateIsNull(resourceData *schema.ResourceData) (result interface{}, err error) {
	// The cty package reports type mismatches by panicking
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(fmt.Sprintf("panic while handling coder_metadata: %#v", r))
		}
	}()

	rawPlan := resourceData.GetRawPlan()
	pairs := rawPlan.GetAttr("pair").AsValueSlice()

	var resultPairs []interface{}
	for _, pair := range pairs {
		resultPair := map[string]interface{}{
			"key":       valueAsString(pair.GetAttr("key")),
			"value":     valueAsString(pair.GetAttr("value")),
			"sensitive": valueAsBool(pair.GetAttr("sensitive")),
		}
		if pair.GetAttr("value").IsNull() {
			resultPair["is_null"] = true
		}
		resultPairs = append(resultPairs, resultPair)
	}

	return resultPairs, nil
}

// valueAsString takes a cty.Value that may be a string or null, and converts it to either a Go string
// or a nil interface{}
func valueAsString(value cty.Value) interface{} {
	if value.IsNull() {
		return ""
	}
	return value.AsString()
}

// valueAsString takes a cty.Value that may be a boolean or null, and converts it to either a Go bool
// or a nil interface{}
func valueAsBool(value cty.Value) interface{} {
	if value.IsNull() {
		return nil
	}
	return value.True()
}

// errorAsDiagnostic transforms a Go error to a diag.Diagnostics object representing a fatal error.
func errorAsDiagnostics(err error) diag.Diagnostics {
	return []diag.Diagnostic{{
		Severity: diag.Error,
		Summary:  err.Error(),
	}}
}
