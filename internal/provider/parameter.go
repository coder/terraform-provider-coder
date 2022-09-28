package provider

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func parameterDataSource() *schema.Resource {
	return &schema.Resource{
		Description: "Use this data source to configure editable options for workspaces.",
		ReadContext: func(ctx context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
			rd.SetId(uuid.NewString())

			name := rd.Get("name").(string)
			typ := rd.Get("type").(string)
			var value string
			rawDefaultValue, ok := rd.GetOk("default")
			if ok {
				defaultValue := rawDefaultValue.(string)
				err := valueIsType(typ, defaultValue)
				if err != nil {
					return err
				}
				value = defaultValue
			}
			envValue, ok := os.LookupEnv(fmt.Sprintf("CODER_PARAMETER_%s", name))
			if ok {
				value = envValue
			}
			rd.Set("value", value)

			rawOptions, exists := rd.GetOk("option")
			if exists {
				rawArrayOptions, valid := rawOptions.([]interface{})
				if !valid {
					return diag.Errorf("options is of wrong type %T", rawArrayOptions)
				}
				optionDisplayNames := map[string]interface{}{}
				optionValues := map[string]interface{}{}
				for _, rawOption := range rawArrayOptions {
					option, valid := rawOption.(map[string]interface{})
					if !valid {
						return diag.Errorf("option is of wrong type %T", rawOption)
					}
					rawName, ok := option["name"]
					if !ok {
						return diag.Errorf("no name for %+v", option)
					}
					displayName, ok := rawName.(string)
					if !ok {
						return diag.Errorf("display name is of wrong type %T", displayName)
					}
					_, exists := optionDisplayNames[displayName]
					if exists {
						return diag.Errorf("multiple options cannot have the same display name %q", displayName)
					}

					rawValue, ok := option["value"]
					if !ok {
						return diag.Errorf("no value for %+v\n", option)
					}
					value, ok := rawValue.(string)
					if !ok {
						return diag.Errorf("")
					}
					_, exists = optionValues[value]
					if exists {
						return diag.Errorf("multiple options cannot have the same value %q", value)
					}
					err := valueIsType(typ, value)
					if err != nil {
						return err
					}

					optionValues[value] = nil
					optionDisplayNames[displayName] = nil
				}
			}

			return nil
		},
		Schema: map[string]*schema.Schema{
			"value": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The output value of a parameter.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the parameter as it appears in the interface. If this is changed, the parameter will need to be re-updated by developers.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Explain what the parameter does.",
			},
			"type": {
				Type:         schema.TypeString,
				Default:      "string",
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"number", "string", "bool"}, false),
			},
			"immutable": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Whether the value can be changed after it's set initially.",
			},
			"default": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "A default value for the parameter.",
				ExactlyOneOf: []string{"option"},
			},
			"icon": {
				Type: schema.TypeString,
				Description: "A URL to an icon that will display in the dashboard. View built-in " +
					"icons here: https://github.com/coder/coder/tree/main/site/static/icon. Use a " +
					"built-in icon with `data.coder_workspace.me.access_url + \"/icon/<path>\"`.",
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
			"option": {
				Type:        schema.TypeList,
				Description: "Each \"option\" block defines a single displayable value for a user to select.",
				ForceNew:    true,
				Optional:    true,
				MaxItems:    64,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Description: "The display name of this value in the UI.",
							ForceNew:    true,
							Required:    true,
						},
						"description": {
							Type:        schema.TypeString,
							Description: "Add a description to select this item.",
							ForceNew:    true,
							Optional:    true,
						},
						"value": {
							Type:        schema.TypeString,
							Description: "The value of this option.",
							ForceNew:    true,
							Required:    true,
						},
						"icon": {
							Type: schema.TypeString,
							Description: "A URL to an icon that will display in the dashboard. View built-in " +
								"icons here: https://github.com/coder/coder/tree/main/site/static/icon. Use a " +
								"built-in icon with `data.coder_workspace.me.access_url + \"/icon/<path>\"`.",
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
					},
				},
			},
			"validation": {
				Type:     schema.TypeSet,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"min": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     0,
							Description: "The minimum for a number to be.",
						},
						"max": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "The maximum for a number to be.",
						},
						"regex": {
							Type:         schema.TypeString,
							ExactlyOneOf: []string{"min", "max"},
							Optional:     true,
						},
					},
				},
			},
		},
	}
}

func valueIsType(typ, value string) diag.Diagnostics {
	switch typ {
	case "number":
		_, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return diag.Errorf("%q is not a number", value)
		}
	case "bool":
		_, err := strconv.ParseBool(value)
		if err != nil {
			return diag.Errorf("%q is not a bool", value)
		}
	case "string":
		// Anything is a string!
	default:
		return diag.Errorf("invalid type %q", typ)
	}
	return nil
}
