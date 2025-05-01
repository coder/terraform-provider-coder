package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/xerrors"
)

type ValidationMode string

const (
	ValidationModeEnvVar = "CODER_VALIDATION_MODE"
	// ValidationModeDefault is used for creating a workspace. It validates the final
	// value used for a parameter. Some allowances for invalid options are tolerated,
	// as unused options do not affect the final parameter value. The default value
	// is also ignored, assuming a value is provided.
	ValidationModeDefault ValidationMode = ""
	// ValidationModeTemplateImport tolerates empty values, as the value might not be
	// available at import. It does not tolerate an invalid default or invalid option
	// values.
	ValidationModeTemplateImport ValidationMode = "template-import"
)

var (
	defaultValuePath = cty.Path{cty.GetAttrStep{Name: "default"}}
)

type Option struct {
	Name        string
	Description string
	Value       string
	Icon        string
}

type Validation struct {
	Min         int
	MinDisabled bool `mapstructure:"min_disabled"`
	Max         int
	MaxDisabled bool `mapstructure:"max_disabled"`

	Monotonic string

	Regex string
	Error string
}

const (
	ValidationMonotonicIncreasing = "increasing"
	ValidationMonotonicDecreasing = "decreasing"
)

type Parameter struct {
	Name        string
	DisplayName string `mapstructure:"display_name"`
	Description string
	Type        OptionType
	FormType    ParameterFormType
	Mutable     bool
	Default     *string
	Icon        string
	Option      []Option
	Validation  []Validation
	Optional    bool
	Order       int
	Ephemeral   bool
}

func parameterDataSource() *schema.Resource {
	return &schema.Resource{
		SchemaVersion: 1,

		Description: "Use this data source to configure editable options for workspaces.",
		ReadContext: func(ctx context.Context, rd *schema.ResourceData, i interface{}) diag.Diagnostics {
			rd.SetId(uuid.NewString())

			fixedValidation, err := fixValidationResourceData(rd.GetRawConfig(), rd.Get("validation"))
			if err != nil {
				return diag.FromErr(err)
			}

			err = rd.Set("validation", fixedValidation)
			if err != nil {
				return diag.FromErr(err)
			}

			var parameter Parameter
			err = mapstructure.Decode(struct {
				Name        interface{}
				DisplayName interface{}
				Description interface{}
				Type        interface{}
				FormType    interface{}
				Mutable     interface{}
				Default     interface{}
				Icon        interface{}
				Option      interface{}
				Validation  interface{}
				Optional    interface{}
				Order       interface{}
				Ephemeral   interface{}
			}{
				Name:        rd.Get("name"),
				DisplayName: rd.Get("display_name"),
				Description: rd.Get("description"),
				Type:        rd.Get("type"),
				FormType:    rd.Get("form_type"),
				Mutable:     rd.Get("mutable"),
				Default: func() *string {
					if rd.GetRawConfig().AsValueMap()["default"].IsNull() {
						return nil
					}
					val, _ := rd.Get("default").(string)
					return &val
				}(),
				Icon:       rd.Get("icon"),
				Option:     rd.Get("option"),
				Validation: fixedValidation,
				Optional: func() bool {
					// This hack allows for checking if the "default" field is present in the .tf file.
					// If "default" is missing or is "null", then it means that this field is required,
					// and user must provide a value for it.
					val := !rd.GetRawConfig().AsValueMap()["default"].IsNull()
					rd.Set("optional", val)
					return val
				}(),
				Order:     rd.Get("order"),
				Ephemeral: rd.Get("ephemeral"),
			}, &parameter)
			if err != nil {
				return diag.Errorf("decode parameter: %s", err)
			}

			if !parameter.Mutable && parameter.Ephemeral {
				return diag.Errorf("parameter can't be immutable and ephemeral")
			}

			if !parameter.Optional && parameter.Ephemeral {
				return diag.Errorf("ephemeral parameter requires the default property")
			}

			var input *string
			envValue, ok := os.LookupEnv(ParameterEnvironmentVariable(parameter.Name))
			if ok {
				input = &envValue
			}

			mode := os.Getenv(ValidationModeEnvVar)
			value, diags := parameter.Valid(input, ValidationMode(mode))
			if diags.HasError() {
				return diags
			}

			// Always set back the value, as it can be sourced from the default
			rd.Set("value", value)

			// Set the form_type as it could have changed in the validation.
			rd.Set("form_type", parameter.FormType)

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
				Description: "The name of the parameter. If this is changed, developers will be re-prompted for a new value.",
			},
			"display_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The displayed name of the parameter as it will appear in the interface.",
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
				ValidateFunc: validation.StringInSlice(toStrings(OptionTypes()), false),
				Description:  "The type of this parameter. Must be one of: `\"number\"`, `\"string\"`, `\"bool\"`, or `\"list(string)\"`.",
			},
			"form_type": {
				Type:         schema.TypeString,
				Default:      ParameterFormTypeDefault,
				Optional:     true,
				ValidateFunc: validation.StringInSlice(toStrings(ParameterFormTypes()), false),
				Description:  fmt.Sprintf("The type of this parameter. Must be one of: [%s].", strings.Join(toStrings(ParameterFormTypes()), ", ")),
			},
			"styling": {
				Type:    schema.TypeString,
				Default: `{}`,
				Description: "JSON encoded string containing the metadata for controlling the appearance of this parameter in the UI. " +
					"This option is purely cosmetic and does not affect the function of the parameter in terraform.",
				Optional: true,
			},
			"mutable": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether this value can be changed after workspace creation. This can be destructive for values like region, so use with caution!",
			},
			"default": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A default value for the parameter.",
			},
			"icon": {
				Type: schema.TypeString,
				Description: "A URL to an icon that will display in the dashboard. View built-in " +
					"icons [here](https://github.com/coder/coder/tree/main/site/static/icon). Use a " +
					"built-in icon with `\"${data.coder_workspace.me.access_url}/icon/<path>\"`.",
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
				Description: "Each `option` block defines a value for a user to select from.",
				ForceNew:    true,
				Optional:    true,
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
								"icons [here](https://github.com/coder/coder/tree/main/site/static/icon). Use a " +
								"built-in icon with `\"${data.coder_workspace.me.access_url}/icon/<path>\"`.",
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
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Validate the input of a parameter.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"min": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "The minimum of a number parameter.",
						},
						"min_disabled": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Helper field to check if min is present",
						},
						"max": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "The maximum of a number parameter.",
						},
						"max_disabled": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Helper field to check if max is present",
						},
						"monotonic": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Number monotonicity, either increasing or decreasing.",
						},
						"regex": {
							Type:          schema.TypeString,
							ConflictsWith: []string{"validation.0.min", "validation.0.max", "validation.0.monotonic"},
							Description:   "A regex for the input parameter to match against.",
							Optional:      true,
						},
						"error": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "An error message to display if the value breaks the validation rules. The following placeholders are supported: {max}, {min}, and {value}.",
						},
					},
				},
			},
			"optional": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether this value is optional.",
			},
			"order": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The order determines the position of a template parameter in the UI/CLI presentation. The lowest order is shown first and parameters with equal order are sorted by name (ascending order).",
			},
			"ephemeral": {
				Type:        schema.TypeBool,
				Default:     false,
				Optional:    true,
				Description: "The value of an ephemeral parameter will not be preserved between consecutive workspace builds.",
			},
		},
	}
}

func fixValidationResourceData(rawConfig cty.Value, validation interface{}) (interface{}, error) {
	// Read validation from raw config
	rawValidation, ok := rawConfig.AsValueMap()["validation"]
	if !ok {
		return validation, nil // no validation rules, nothing to fix
	}

	rawValidationArr := rawValidation.AsValueSlice()
	if len(rawValidationArr) == 0 {
		return validation, nil // no validation rules, nothing to fix
	}

	rawValidationRule := rawValidationArr[0].AsValueMap()

	// Load validation from resource data
	vArr, ok := validation.([]interface{})
	if !ok {
		return nil, xerrors.New("validation should be an array")
	}

	if len(vArr) == 0 {
		return validation, nil // no validation rules, nothing to fix
	}

	validationRule, ok := vArr[0].(map[string]interface{})
	if !ok {
		return nil, xerrors.New("validation rule should be a map")
	}

	validationRule["min_disabled"] = rawValidationRule["min"].IsNull()
	validationRule["max_disabled"] = rawValidationRule["max"].IsNull()
	return vArr, nil
}

func valueIsType(typ OptionType, value string) error {
	switch typ {
	case OptionTypeNumber:
		_, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("%q is not a number", value)
		}
	case OptionTypeBoolean:
		_, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("%q is not a bool", value)
		}
	case OptionTypeListString:
		_, err := valueIsListString(value)
		if err != nil {
			return err
		}
	case OptionTypeString:
		// Anything is a string!
	default:
		return fmt.Errorf("invalid type %q", typ)
	}
	return nil
}

func (v *Parameter) Valid(input *string, mode ValidationMode) (string, diag.Diagnostics) {
	var err error
	var optionType OptionType

	value := input
	if input == nil {
		value = v.Default
	}

	// TODO: When empty values want to be rejected, uncomment this.
	//   coder/coder should update to use template import mode first,
	//   before this is uncommented.
	//if value == nil && mode == ValidationModeDefault {
	//	var empty string
	//	value = &empty
	//}

	// optionType might differ from parameter.Type. This is ok, and parameter.Type
	// should be used for the value type, and optionType for options.
	optionType, v.FormType, err = ValidateFormType(v.Type, len(v.Option), v.FormType)
	if err != nil {
		return "", diag.Diagnostics{
			{
				Severity:      diag.Error,
				Summary:       "Invalid form_type for parameter",
				Detail:        err.Error(),
				AttributePath: cty.Path{cty.GetAttrStep{Name: "form_type"}},
			},
		}
	}

	optionValues, diags := v.ValidOptions(optionType, mode)
	if diags.HasError() {
		return "", diags
	}

	if mode == ValidationModeTemplateImport && v.Default != nil {
		// Template import should validate the default value.
		err := valueIsType(v.Type, *v.Default)
		if err != nil {
			return "", diag.Diagnostics{
				{
					Severity:      diag.Error,
					Summary:       fmt.Sprintf("Default value is not of type %q", v.Type),
					Detail:        err.Error(),
					AttributePath: defaultValuePath,
				},
			}
		}

		d := v.validValue(*v.Default, optionType, optionValues, defaultValuePath)
		if d.HasError() {
			return "", d
		}
	}

	// TODO: Move this into 'Parameter.validValue'. It exists as another check outside because
	// the current behavior is to always apply this validation, regardless if the param is set or not.
	// Other validations are only applied if the parameter is set.
	// This behavior should be standardized.
	if len(v.Validation) == 1 {
		empty := ""
		validVal := value
		if value == nil {
			validVal = &empty
		}
		validCheck := &v.Validation[0]
		err := validCheck.Valid(v.Type, *validVal)
		if err != nil {
			return "", diag.Diagnostics{
				{
					Severity:      diag.Error,
					Summary:       fmt.Sprintf("Invalid parameter %s according to 'validation' block", strings.ToLower(v.Name)),
					Detail:        err.Error(),
					AttributePath: cty.Path{},
				},
			}
		}
	}

	// TODO: This is a bit of a hack. The current behavior states if validation
	//   is given, then apply validation to unset values.
	//   This should be removed, and all values should be validated. Meaning
	//   value == nil should not be accepted in the first place.
	if len(v.Validation) > 0 && value == nil {
		empty := ""
		value = &empty
	}

	// Value is only validated if it is set. If it is unset, validation
	// is skipped.
	if value != nil {
		d := v.validValue(*value, optionType, optionValues, cty.Path{})
		if d.HasError() {
			return "", d
		}

		err = valueIsType(v.Type, *value)
		if err != nil {
			return "", diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  fmt.Sprintf("Parameter value is not of type %q", v.Type),
					Detail:   err.Error(),
				},
			}
		}
	}

	if value == nil {
		// The previous behavior is to always write an empty string
		return "", nil
	}

	return *value, nil
}

func (v *Parameter) ValidOptions(optionType OptionType, mode ValidationMode) (map[string]struct{}, diag.Diagnostics) {
	optionNames := map[string]struct{}{}
	optionValues := map[string]struct{}{}

	var diags diag.Diagnostics
	for _, option := range v.Option {
		_, exists := optionNames[option.Name]
		if exists {
			return nil, diag.Diagnostics{{
				Severity: diag.Error,
				Summary:  "Option names must be unique.",
				Detail:   fmt.Sprintf("multiple options found with the same name %q", option.Name),
			}}
		}

		_, exists = optionValues[option.Value]
		if exists {
			return nil, diag.Diagnostics{{
				Severity: diag.Error,
				Summary:  "Option values must be unique.",
				Detail:   fmt.Sprintf("multiple options found with the same value %q", option.Value),
			}}
		}

		err := valueIsType(optionType, option.Value)
		if err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("Option %q with value=%q is not of type %q", option.Name, option.Value, optionType),
				Detail:   err.Error(),
			})
			continue
		}
		optionValues[option.Value] = struct{}{}
		optionNames[option.Name] = struct{}{}

		if mode == ValidationModeTemplateImport {
			opDiags := v.validValue(option.Value, optionType, nil, cty.Path{})
			if opDiags.HasError() {
				for i := range opDiags {
					opDiags[i].Summary = fmt.Sprintf("Option %q: %s", option.Name, opDiags[i].Summary)
				}
				diags = append(diags, opDiags...)
			}
		}
	}

	if diags.HasError() {
		return nil, diags
	}
	return optionValues, nil
}

func (v *Parameter) validValue(value string, optionType OptionType, optionValues map[string]struct{}, path cty.Path) diag.Diagnostics {
	// name is used for constructing more precise error messages.
	name := "Value"
	if path.Equals(defaultValuePath) {
		name = "Default value"
	}

	// First validate if the value is a valid option
	if len(optionValues) > 0 {
		if v.Type == OptionTypeListString && optionType == OptionTypeString {
			// If the type is list(string) and optionType is string, we have
			// to ensure all elements of the default exist as options.
			listValues, err := valueIsListString(value)
			if err != nil {
				return diag.Diagnostics{
					{
						Severity:      diag.Error,
						Summary:       "When using list(string) type, value must be a json encoded list of strings",
						Detail:        err.Error(),
						AttributePath: defaultValuePath,
					},
				}
			}

			// missing is used to construct a more helpful error message
			var missing []string
			for _, listValue := range listValues {
				_, isValid := optionValues[listValue]
				if !isValid {
					missing = append(missing, listValue)
				}
			}

			if len(missing) > 0 {
				return diag.Diagnostics{
					{
						Severity: diag.Error,
						Summary:  fmt.Sprintf("%ss must be a valid option", name),
						Detail: fmt.Sprintf(
							"%s %q is not a valid option, values %q are missing from the options",
							name, value, strings.Join(missing, ", "),
						),
						AttributePath: defaultValuePath,
					},
				}
			}
		} else {
			_, isValid := optionValues[value]
			if !isValid {
				extra := ""
				if value == "" {
					extra = ". The value is empty, did you forget to set it with a default or from user input?"
				}
				return diag.Diagnostics{
					{
						Severity:      diag.Error,
						Summary:       fmt.Sprintf("%s must be a valid option%s", name, extra),
						Detail:        fmt.Sprintf("the value %q must be defined as one of options", value),
						AttributePath: path,
					},
				}
			}
		}
	}

	if len(v.Validation) == 1 {
		validCheck := &v.Validation[0]
		err := validCheck.Valid(v.Type, value)
		if err != nil {
			return diag.Diagnostics{
				{
					Severity:      diag.Error,
					Summary:       fmt.Sprintf("Invalid parameter %s according to 'validation' block", strings.ToLower(name)),
					Detail:        err.Error(),
					AttributePath: path,
				},
			}
		}
	}

	return nil
}

func (v *Validation) Valid(typ OptionType, value string) error {
	if typ != OptionTypeNumber {
		if !v.MinDisabled {
			return fmt.Errorf("a min cannot be specified for a %s type", typ)
		}
		if !v.MaxDisabled {
			return fmt.Errorf("a max cannot be specified for a %s type", typ)
		}
		if v.Monotonic != "" {
			return fmt.Errorf("monotonic validation can only be specified for number types, not %s types", typ)
		}
	}
	if typ != OptionTypeString && v.Regex != "" {
		return fmt.Errorf("a regex cannot be specified for a %s type", typ)
	}
	switch typ {
	case OptionTypeBoolean:
		if value != "true" && value != "false" {
			return fmt.Errorf(`boolean value can be either "true" or "false"`)
		}
		return nil
	case OptionTypeString:
		if v.Regex == "" {
			return nil
		}
		regex, err := regexp.Compile(v.Regex)
		if err != nil {
			return fmt.Errorf("compile regex %q: %s", regex, err)
		}
		if v.Error == "" {
			return fmt.Errorf("an error must be specified with a regex validation")
		}
		matched := regex.MatchString(value)
		if !matched {
			return fmt.Errorf("%s (value %q does not match %q)", v.Error, value, regex)
		}
	case OptionTypeNumber:
		num, err := strconv.Atoi(value)
		if err != nil {
			return takeFirstError(v.errorRendered(value), fmt.Errorf("value %q is not a number", value))
		}
		if !v.MinDisabled && num < v.Min {
			return takeFirstError(v.errorRendered(value), fmt.Errorf("value %d is less than the minimum %d", num, v.Min))
		}
		if !v.MaxDisabled && num > v.Max {
			return takeFirstError(v.errorRendered(value), fmt.Errorf("value %d is more than the maximum %d", num, v.Max))
		}
		if v.Monotonic != "" && v.Monotonic != ValidationMonotonicIncreasing && v.Monotonic != ValidationMonotonicDecreasing {
			return fmt.Errorf("number monotonicity can be either %q or %q", ValidationMonotonicIncreasing, ValidationMonotonicDecreasing)
		}
	case OptionTypeListString:
		var listOfStrings []string
		err := json.Unmarshal([]byte(value), &listOfStrings)
		if err != nil {
			return fmt.Errorf("value %q is not valid list of strings", value)
		}
	}
	return nil
}

func valueIsListString(value string) ([]string, error) {
	var items []string
	err := json.Unmarshal([]byte(value), &items)
	if err != nil {
		return nil, fmt.Errorf("value %q is not a valid list of strings", value)
	}
	return items, nil
}

// ParameterEnvironmentVariable returns the environment variable to specify for
// a parameter by it's name. It's hashed because spaces and special characters
// can be used in parameter names that may not be valid in env vars.
func ParameterEnvironmentVariable(name string) string {
	sum := sha256.Sum256([]byte(name))
	return "CODER_PARAMETER_" + hex.EncodeToString(sum[:])
}

func takeFirstError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return xerrors.Errorf("developer error: error message is not provided")
}

func (v *Validation) errorRendered(value string) error {
	if v.Error == "" {
		return nil
	}
	r := strings.NewReplacer(
		"{min}", fmt.Sprintf("%d", v.Min),
		"{max}", fmt.Sprintf("%d", v.Max),
		"{value}", value)
	return xerrors.Errorf(r.Replace(v.Error))
}
