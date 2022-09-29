package provider

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"regexp"
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

			rawValidation, exists := rd.GetOk("validation")
			var (
				validationRegex string
				validationMin   int
				validationMax   int
			)
			if exists {
				validationArray, valid := rawValidation.([]interface{})
				if !valid {
					return diag.Errorf("validation is of wrong type %T", rawValidation)
				}
				validation, valid := validationArray[0].(map[string]interface{})
				if !valid {
					return diag.Errorf("validation is of wrong type %T", validation)
				}
				rawRegex, ok := validation["regex"]
				if ok {
					validationRegex, ok = rawRegex.(string)
					if !ok {
						return diag.Errorf("validation regex is of wrong type %T", rawRegex)
					}
				}
				rawMin, ok := validation["min"]
				if ok {
					validationMin, ok = rawMin.(int)
					if !ok {
						return diag.Errorf("validation min is wrong type %T", rawMin)
					}
				}
				rawMax, ok := validation["max"]
				if ok {
					validationMax, ok = rawMax.(int)
					if !ok {
						return diag.Errorf("validation max is wrong type %T", rawMax)
					}
				}
			}

			err := ValueValidatesType(typ, value, validationRegex, validationMin, validationMax)
			if err != nil {
				return diag.FromErr(err)
			}

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
				Description: "The output value of the parameter.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the parameter as it will appear in the interface. If this is changed, developers will be re-prompted for a new value.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Describe what this parameter does.",
			},
			"type": {
				Type:         schema.TypeString,
				Default:      "string",
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"number", "string", "bool"}, false),
				Description:  `The type of this parameter. Must be one of: "number", "string", or "bool".`,
			},
			"immutable": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether this value can be changed after workspace creation. This can be destructive for values like region, so use with caution!",
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
				Type:          schema.TypeList,
				Description:   "Each \"option\" block defines a value for a user to select from.",
				ForceNew:      true,
				Optional:      true,
				MaxItems:      64,
				ConflictsWith: []string{"validation"},
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
							Description: "Describe what selecting this value does.",
							ForceNew:    true,
							Optional:    true,
						},
						"value": {
							Type:        schema.TypeString,
							Description: "The value of this option set on the parameter if selected.",
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
				Type:          schema.TypeList,
				MaxItems:      1,
				Optional:      true,
				Description:   "Validate the input of a parameter.",
				ConflictsWith: []string{"option"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"min": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      0,
							Description:  "The minimum of a number parameter.",
							RequiredWith: []string{"validation.0.max"},
						},
						"max": {
							Type:         schema.TypeInt,
							Optional:     true,
							Description:  "The maximum of a number parameter.",
							RequiredWith: []string{"validation.0.min"},
						},
						"regex": {
							Type:          schema.TypeString,
							ConflictsWith: []string{"validation.0.min", "validation.0.max"},
							Description:   "A regex for the input parameter to match against.",
							Optional:      true,
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

func ValueValidatesType(typ, value, regex string, min, max int) error {
	if typ != "number" {
		if min != 0 {
			return fmt.Errorf("a min cannot be specified for a %s type", typ)
		}
		if max != 0 {
			return fmt.Errorf("a max cannot be specified for a %s type", typ)
		}
	}
	if typ != "string" && regex != "" {
		return fmt.Errorf("a regex cannot be specified for a %s type", typ)
	}
	switch typ {
	case "bool":
		return nil
	case "string":
		if regex == "" {
			return nil
		}
		regex, err := regexp.Compile(regex)
		if err != nil {
			return fmt.Errorf("compile regex %q: %s", regex, err)
		}
		matched := regex.MatchString(value)
		if !matched {
			return fmt.Errorf("value %q does not match %q", value, regex)
		}
	case "number":
		num, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("parse value %s as int: %s", value, err)
		}
		if num < min {
			return fmt.Errorf("provided value %d is less than the minimum %d", num, min)
		}
		if num > max {
			return fmt.Errorf("provided value %d is more than the maximum %d", num, max)
		}
	}
	return nil
}
