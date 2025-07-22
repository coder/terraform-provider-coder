package provider

import (
	"context"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"golang.org/x/xerrors"

	"github.com/coder/terraform-provider-coder/v2/provider/helpers"
)

func metadataResource() *schema.Resource {
	return &schema.Resource{
		SchemaVersion: 1,

		Description: "Use this resource to attach metadata to a resource. They will be " +
			"displayed in the Coder dashboard alongside the resource. " +
			"The resource containing the agent, and it's metadata, will be shown by default. " + "\n\n" +
			"Alternatively, to attach metadata to the agent, use a `metadata` block within a `coder_agent` resource.",
		CreateContext: func(c context.Context, resourceData *schema.ResourceData, i interface{}) diag.Diagnostics {
			resourceData.SetId(uuid.NewString())

			items, err := populateIsNull(resourceData)
			if err != nil {
				return errorAsDiagnostics(err)
			}
			err = resourceData.Set("item", items)
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
				Description: "The `id` property of another resource that metadata should be attached to.",
				ForceNew:    true,
				Required:    true,
			},
			"hide": {
				Type:        schema.TypeBool,
				Description: "Hide the resource from the UI.",
				ForceNew:    true,
				Optional:    true,
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
			"daily_cost": {
				Type: schema.TypeInt,
				Description: "(Enterprise) The cost of this resource every 24 hours." +
					" Use the smallest denomination of your preferred currency." +
					" For example, if you work in USD, use cents.",
				ForceNew: true,
				Optional: true,
			},
			"item": {
				Type:        schema.TypeList,
				Description: "Each `item` block defines a single metadata item consisting of a key/value pair.",
				ForceNew:    true,
				Optional:    true,
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
							Description: "The value of this metadata item. Supports basic Markdown, including hyperlinks.",
							ForceNew:    true,
							Optional:    true,
						},
						"sensitive": {
							Type: schema.TypeBool,
							Description: "Set to `true` to for items such as API keys whose values should be " +
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
		CustomizeDiff: func(ctx context.Context, rd *schema.ResourceDiff, i interface{}) error {
			if !rd.HasChange("item") {
				return nil
			}

			keys := map[string]bool{}
			metadata, ok := rd.Get("item").([]any)
			if !ok {
				return xerrors.Errorf("unexpected type %T for items, expected []any", rd.Get("metadata"))
			}
			for _, t := range metadata {
				obj, ok := t.(map[string]any)
				if !ok {
					return xerrors.Errorf("unexpected type %T for item, expected map[string]any", t)
				}
				key, ok := obj["key"].(string)
				if !ok {
					return xerrors.Errorf("unexpected type %T for items 'key' attribute, expected string", obj["key"])
				}
				if keys[key] {
					return xerrors.Errorf("duplicate resource metadata key %q", key)
				}
				keys[key] = true
			}
			return nil
		},
	}
}
