package provider_test

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/coder/terraform-provider-coder/v2/provider"
)

func TestParameter(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		Name        string
		Config      string
		ExpectError *regexp.Regexp
		Check       func(state *terraform.ResourceState)
	}{{
		Name: "FieldsExist",
		Config: `
			data "coder_parameter" "region" {
				name = "region"
				display_name = "Region"
				type = "string"
 				form_type = "dropdown"
				description = <<-EOT
					# Select the machine image
					See the [registry](https://container.registry.blah/namespace) for options.
					EOT
				mutable = true
				icon = "/icon/region.svg"
				default = "us-east1-a"
				option {
					name = "US Central"
					value = "us-central1-a"
					icon = "/icon/central.svg"
					description = "Select for central!"
				}
				option {
					name = "US East"
					value = "us-east1-a"
					icon = "/icon/east.svg"
					description = "Select for east!"
				}
				order = 5
				ephemeral = true
			}
			`,
		Check: func(state *terraform.ResourceState) {
			attrs := state.Primary.Attributes
			for key, value := range map[string]interface{}{
				"name":                 "region",
				"display_name":         "Region",
				"type":                 "string",
				"form_type":            "dropdown",
				"description":          "# Select the machine image\nSee the [registry](https://container.registry.blah/namespace) for options.\n",
				"mutable":              "true",
				"icon":                 "/icon/region.svg",
				"option.0.name":        "US Central",
				"option.0.value":       "us-central1-a",
				"option.0.icon":        "/icon/central.svg",
				"option.0.description": "Select for central!",
				"option.1.name":        "US East",
				"option.1.value":       "us-east1-a",
				"option.1.icon":        "/icon/east.svg",
				"option.1.description": "Select for east!",
				"order":                "5",
				"default":              "us-east1-a",
				"ephemeral":            "true",
			} {
				require.Equal(t, value, attrs[key])
			}
		},
	}, {
		Name: "RegexValidationWithOptions",
		Config: `
			data "coder_parameter" "region" {
				name = "Region"
				type = "number"
				default = 1
				option {
					name = "1"
					value = "1"
				}
				validation {
					regex = "1"
					error = "Not 1"
				}
			}
			`,
		ExpectError: regexp.MustCompile("a regex cannot be specified for a number type"),
	}, {
		Name: "MonotonicValidationWithNonNumberType",
		Config: `
			data "coder_parameter" "region" {
				name = "Region"
				type = "string"
				default = "1"
				option {
					name = "1"
					value = "1"
				}
				validation {
					monotonic = "increasing"
				}
			}
			`,
		ExpectError: regexp.MustCompile("monotonic validation can only be specified for number types, not string types"),
	}, {
		Name: "ValidationRegexMissingError",
		Config: `
			data "coder_parameter" "region" {
				name = "Region"
				type = "string"
				default = "hello"
				validation {
					regex = "hello"
				}
			}
			`,
		ExpectError: regexp.MustCompile("an error must be specified"),
	}, {
		Name: "NumberValidation",
		Config: `
			data "coder_parameter" "region" {
				name = "Region"
				type = "number"
				default = 2
				validation {
					min = 1
					max = 5
				}
			}
			`,
		Check: func(state *terraform.ResourceState) {
			for key, expected := range map[string]string{
				"name":                      "Region",
				"type":                      "number",
				"form_type":                 "input",
				"validation.#":              "1",
				"default":                   "2",
				"validation.0.min":          "1",
				"validation.0.max":          "5",
				"validation.0.min_disabled": "false",
				"validation.0.max_disabled": "false",
			} {
				require.Equal(t, expected, state.Primary.Attributes[key])
			}
		},
	}, {
		Name: "DefaultNotNumber",
		Config: `
			data "coder_parameter" "region" {
				name = "Region"
				type = "number"
				default = true
			}
			`,
		ExpectError: regexp.MustCompile("is not a number"),
	}, {
		Name: "DefaultNotBool",
		Config: `
			data "coder_parameter" "region" {
				name = "Region"
				type = "bool"
				default = 5
			}
			`,
		ExpectError: regexp.MustCompile("is not a bool"),
	}, {
		Name: "OptionNotBool",
		Config: `
			data "coder_parameter" "region" {
				name = "Region"
				type = "bool"
				option {
					value = 1
					name = 1
				}
				option {
					value = 2
					name = 2
				}
			}`,
		ExpectError: regexp.MustCompile("\"2\" is not a bool"),
	}, {
		Name: "MultipleOptions",
		Config: `
			data "coder_parameter" "region" {
				name = "Region"
				type = "string"
				option {
					name = "1"
					value = "1"
					icon = "/icon/code.svg"
					description = "Something!"
				}
				option {
					name = "2"
					value = "2"
				}
			}
			`,
		Check: func(state *terraform.ResourceState) {
			for key, expected := range map[string]string{
				"name":                 "Region",
				"option.#":             "2",
				"option.0.name":        "1",
				"option.0.value":       "1",
				"option.0.icon":        "/icon/code.svg",
				"option.0.description": "Something!",
			} {
				require.Equal(t, expected, state.Primary.Attributes[key])
			}
		},
	}, {
		Name: "ValidDefaultWithOptions",
		Config: `
			data "coder_parameter" "region" {
				name = "Region"
				type = "string"
				default = "2"
				option {
					name = "1"
					value = "1"
					icon = "/icon/code.svg"
					description = "Something!"
				}
				option {
					name = "2"
					value = "2"
				}
			}
			`,
		Check: func(state *terraform.ResourceState) {
			for key, expected := range map[string]string{
				"name":                 "Region",
				"option.#":             "2",
				"option.0.name":        "1",
				"option.0.value":       "1",
				"option.0.icon":        "/icon/code.svg",
				"option.0.description": "Something!",
			} {
				require.Equal(t, expected, state.Primary.Attributes[key])
			}
		},
	}, {
		Name: "InvalidDefaultWithOption",
		Config: `
			data "coder_parameter" "region" {
				name = "Region"
				default = "hi"
				option {
					name = "1"
					value = "1"
				}
				option {
					name = "2"
					value = "2"
				}
			}
			`,
		ExpectError: regexp.MustCompile("must be defined as one of options"),
	}, {
		Name: "SingleOption",
		Config: `
			data "coder_parameter" "region" {
				name = "Region"
				option {
					name = "1"
					value = "1"
				}
			}
			`,
	}, {
		Name: "DuplicateOptionName",
		Config: `
			data "coder_parameter" "region" {
				name = "Region"
				type = "string"
				option {
					name = "1"
					value = "1"
				}
				option {
					name = "1"
					value = "2"
				}
			}
			`,
		ExpectError: regexp.MustCompile("Option names must be unique"),
	}, {
		Name: "DuplicateOptionValue",
		Config: `
			data "coder_parameter" "region" {
				name = "Region"
				type = "string"
				option {
					name = "1"
					value = "1"
				}
				option {
					name = "2"
					value = "1"
				}
			}
			`,
		ExpectError: regexp.MustCompile("Option values must be unique"),
	}, {
		Name: "RequiredParameterNoDefault",
		Config: `
			data "coder_parameter" "region" {
				name = "Region"
				type = "string"
			}`,
		Check: func(state *terraform.ResourceState) {
			for key, expected := range map[string]string{
				"name":     "Region",
				"type":     "string",
				"optional": "false",
			} {
				require.Equal(t, expected, state.Primary.Attributes[key])
			}
		},
	}, {
		Name: "RequiredParameterDefaultNull",
		Config: `
			data "coder_parameter" "region" {
				name = "Region"
				type = "string"
				default = null
			}`,
		Check: func(state *terraform.ResourceState) {
			for key, expected := range map[string]string{
				"name":     "Region",
				"type":     "string",
				"optional": "false",
			} {
				require.Equal(t, expected, state.Primary.Attributes[key])
			}
		},
	}, {
		Name: "OptionalParameterDefaultEmpty",
		Config: `
			data "coder_parameter" "region" {
				name = "Region"
				type = "string"
				default = ""
			}`,
		Check: func(state *terraform.ResourceState) {
			for key, expected := range map[string]string{
				"name":     "Region",
				"type":     "string",
				"optional": "true",
			} {
				require.Equal(t, expected, state.Primary.Attributes[key])
			}
		},
	}, {
		Name: "OptionalParameterDefaultNotEmpty",
		Config: `
			data "coder_parameter" "region" {
				name = "Region"
				type = "string"
				default = "us-east-1"
			}`,
		Check: func(state *terraform.ResourceState) {
			for key, expected := range map[string]string{
				"name":     "Region",
				"type":     "string",
				"optional": "true",
			} {
				require.Equal(t, expected, state.Primary.Attributes[key])
			}
		},
	}, {
		Name: "ListOfStrings",
		Config: `
data "coder_parameter" "region" {
	name = "Region"
	type = "list(string)"
	default = jsonencode(["us-east-1", "eu-west-1", "ap-northeast-1"])
}`,
		Check: func(state *terraform.ResourceState) {
			for key, expected := range map[string]string{
				"name":    "Region",
				"type":    "list(string)",
				"default": `["us-east-1","eu-west-1","ap-northeast-1"]`,
			} {
				attributeValue, ok := state.Primary.Attributes[key]
				require.True(t, ok, "attribute %q is expected", key)
				require.Equal(t, expected, attributeValue)
			}
		},
	}, {
		Name: "NumberValidation_Max",
		Config: `
			data "coder_parameter" "region" {
				name = "Region"
				type = "number"
				default = 2
				validation {
					max = 9
				}
			}
			`,
		Check: func(state *terraform.ResourceState) {
			for key, expected := range map[string]string{
				"name":                      "Region",
				"type":                      "number",
				"validation.#":              "1",
				"default":                   "2",
				"validation.0.max":          "9",
				"validation.0.min_disabled": "true",
				"validation.0.max_disabled": "false",
			} {
				require.Equal(t, expected, state.Primary.Attributes[key])
			}
		},
	}, {
		Name: "NumberValidation_MaxZero",
		Config: `
			data "coder_parameter" "region" {
				name = "Region"
				type = "number"
				default = -1
				validation {
					max = 0
				}
			}
			`,
		Check: func(state *terraform.ResourceState) {
			for key, expected := range map[string]string{
				"name":                      "Region",
				"type":                      "number",
				"validation.#":              "1",
				"default":                   "-1",
				"validation.0.max":          "0",
				"validation.0.min_disabled": "true",
				"validation.0.max_disabled": "false",
			} {
				require.Equal(t, expected, state.Primary.Attributes[key])
			}
		},
	}, {
		Name: "NumberValidation_MonotonicWithOptions",
		Config: `
			data "coder_parameter" "region" {
			  name        = "Region"
			  type        = "number"
			  description = <<-EOF
			  Always pick a larger region.
			  EOF
			  default     = 1

			  option {
				name  = "Small"
				value = 1
			  }

			  option {
				name  = "Medium"
				value = 2
			  }

			  option {
				name  = "Large"
				value = 3
			  }

			  validation {
				monotonic = "increasing"
			  }
			}
			`,
		Check: func(state *terraform.ResourceState) {
			for key, expected := range map[string]any{
				"name":                   "Region",
				"type":                   "number",
				"validation.#":           "1",
				"option.0.name":          "Small",
				"option.0.value":         "1",
				"option.1.name":          "Medium",
				"option.1.value":         "2",
				"option.2.name":          "Large",
				"option.2.value":         "3",
				"default":                "1",
				"validation.0.monotonic": "increasing",
			} {
				require.Equal(t, expected, state.Primary.Attributes[key])
			}
		},
	}, {
		Name: "NumberValidation_Min",
		Config: `
			data "coder_parameter" "region" {
				name = "Region"
				type = "number"
				default = 2
				validation {
					min = 1
				}
			}
			`,
		Check: func(state *terraform.ResourceState) {
			for key, expected := range map[string]string{
				"name":                      "Region",
				"type":                      "number",
				"validation.#":              "1",
				"default":                   "2",
				"validation.0.min":          "1",
				"validation.0.min_disabled": "false",
				"validation.0.max_disabled": "true",
			} {
				require.Equal(t, expected, state.Primary.Attributes[key])
			}
		},
	}, {
		Name: "NumberValidation_MinZero",
		Config: `
			data "coder_parameter" "region" {
				name = "Region"
				type = "number"
				default = 2
				validation {
					min = 0
				}
			}
			`,
		Check: func(state *terraform.ResourceState) {
			for key, expected := range map[string]string{
				"name":                      "Region",
				"type":                      "number",
				"validation.#":              "1",
				"default":                   "2",
				"validation.0.min":          "0",
				"validation.0.min_disabled": "false",
				"validation.0.max_disabled": "true",
			} {
				require.Equal(t, expected, state.Primary.Attributes[key])
			}
		},
	}, {
		Name: "NumberValidation_MinMaxZero",
		Config: `
			data "coder_parameter" "region" {
				name = "Region"
				type = "number"
				default = 0
				validation {
					max = 0
					min = 0
				}
			}
			`,
		Check: func(state *terraform.ResourceState) {
			for key, expected := range map[string]string{
				"name":                      "Region",
				"type":                      "number",
				"validation.#":              "1",
				"default":                   "0",
				"validation.0.min":          "0",
				"validation.0.max":          "0",
				"validation.0.min_disabled": "false",
				"validation.0.max_disabled": "false",
			} {
				require.Equal(t, expected, state.Primary.Attributes[key])
			}
		},
	}, {
		Name: "NumberValidation_LesserThanMin",
		Config: `
			data "coder_parameter" "region" {
				name = "Region"
				type = "number"
				default = 5
				validation {
					min = 7
				}
			}
			`,
		ExpectError: regexp.MustCompile("is less than the minimum"),
	}, {
		Name: "NumberValidation_GreaterThanMin",
		Config: `
			data "coder_parameter" "region" {
				name = "Region"
				type = "number"
				default = 5
				validation {
					max = 3
				}
			}
			`,
		ExpectError: regexp.MustCompile("is more than the maximum"),
	}, {
		Name: "NumberValidation_CustomError",
		Config: `
			data "coder_parameter" "region" {
				name = "Region"
				type = "number"
				default = 5
				validation {
					max = 3
					error = "foobar"
				}
			}
			`,
		ExpectError: regexp.MustCompile("foobar"),
	}, {
		Name: "NumberValidation_NotInRange",
		Config: `
			data "coder_parameter" "region" {
				name = "Region"
				type = "number"
				default = 8
				validation {
					min = 3
					max = 5
				}
			}
			`,
		ExpectError: regexp.MustCompile("is more than the maximum"),
	}, {
		Name: "NumberValidation_BoolWithMin",
		Config: `
			data "coder_parameter" "region" {
				name = "Region"
				type = "bool"
				default = true
				validation {
					min = 7
				}
			}
			`,
		ExpectError: regexp.MustCompile("a min cannot be specified for a bool type"),
	}, {
		Name: "ImmutableEphemeralError",
		Config: `
			data "coder_parameter" "region" {
				name = "Region"
				type = "string"
				default = "abc"
				mutable = false
				ephemeral = true
			}
			`,
		ExpectError: regexp.MustCompile("parameter can't be immutable and ephemeral"),
	}, {
		Name: "RequiredEphemeralError",
		Config: `
			data "coder_parameter" "region" {
				name = "Region"
				type = "string"
				mutable = true
				ephemeral = true
			}
			`,
		ExpectError: regexp.MustCompile("ephemeral parameter requires the default property"),
	}} {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			resource.Test(t, resource.TestCase{
				ProviderFactories: coderFactory(),
				IsUnitTest:        true,
				Steps: []resource.TestStep{{
					Config:      tc.Config,
					ExpectError: tc.ExpectError,
					Check: func(state *terraform.State) error {
						require.Len(t, state.Modules, 1)
						require.Len(t, state.Modules[0].Resources, 1)
						param := state.Modules[0].Resources["data.coder_parameter.region"]
						require.NotNil(t, param)
						if tc.Check != nil {
							tc.Check(param)
						}
						return nil
					},
				}},
			})
		})
	}
}

func TestParameterValidation(t *testing.T) {
	t.Parallel()
	opts := func(vals ...string) []provider.Option {
		options := make([]provider.Option, 0, len(vals))
		for _, val := range vals {
			options = append(options, provider.Option{
				Name:  val,
				Value: val,
			})
		}
		return options
	}

	for _, tc := range []struct {
		Name        string
		Parameter   provider.Parameter
		Value       string
		ExpectError *regexp.Regexp
	}{
		{
			Name: "ValidStringParameter",
			Parameter: provider.Parameter{
				Type: "string",
			},
			Value: "alpha",
		},
		// Test invalid states
		{
			Name: "InvalidFormType",
			Parameter: provider.Parameter{
				Type:     "string",
				Option:   opts("alpha", "bravo", "charlie"),
				FormType: provider.ParameterFormTypeSlider,
			},
			Value:       "alpha",
			ExpectError: regexp.MustCompile("Invalid form_type for parameter"),
		},
		{
			Name: "NotInOptions",
			Parameter: provider.Parameter{
				Type:   "string",
				Option: opts("alpha", "bravo", "charlie"),
			},
			Value:       "delta", // not in option set
			ExpectError: regexp.MustCompile("Value must be a valid option"),
		},
		{
			Name: "NumberNotInOptions",
			Parameter: provider.Parameter{
				Type:   "number",
				Option: opts("1", "2", "3"),
			},
			Value:       "0", // not in option set
			ExpectError: regexp.MustCompile("Value must be a valid option"),
		},
		{
			Name: "NonUniqueOptionNames",
			Parameter: provider.Parameter{
				Type:   "string",
				Option: opts("alpha", "alpha"),
			},
			Value:       "alpha",
			ExpectError: regexp.MustCompile("Option names must be unique"),
		},
		{
			Name: "NonUniqueOptionValues",
			Parameter: provider.Parameter{
				Type: "string",
				Option: []provider.Option{
					{Name: "Alpha", Value: "alpha"},
					{Name: "AlphaAgain", Value: "alpha"},
				},
			},
			Value:       "alpha",
			ExpectError: regexp.MustCompile("Option values must be unique"),
		},
		{
			Name: "IncorrectValueTypeOption",
			Parameter: provider.Parameter{
				Type:   "number",
				Option: opts("not-a-number"),
			},
			Value:       "5",
			ExpectError: regexp.MustCompile("is not a number"),
		},
		{
			Name: "IncorrectValueType",
			Parameter: provider.Parameter{
				Type: "number",
			},
			Value:       "not-a-number",
			ExpectError: regexp.MustCompile("Parameter value is not of type \"number\""),
		},
		{
			Name: "NotListStringDefault",
			Parameter: provider.Parameter{
				Type:    "list(string)",
				Default: ptr("not-a-list"),
			},
			ExpectError: regexp.MustCompile("not a valid list of strings"),
		},
		{
			Name: "NotListStringDefault",
			Parameter: provider.Parameter{
				Type: "list(string)",
			},
			Value:       "not-a-list",
			ExpectError: regexp.MustCompile("not a valid list of strings"),
		},
		{
			Name: "DefaultListStringNotInOptions",
			Parameter: provider.Parameter{
				Type:     "list(string)",
				Default:  ptr(`["red", "yellow", "black"]`),
				Option:   opts("red", "blue", "green"),
				FormType: provider.ParameterFormTypeMultiSelect,
			},
			Value:       `["red", "yellow", "black"]`,
			ExpectError: regexp.MustCompile("is not a valid option, values \"yellow, black\" are missing from the options"),
		},
		{
			Name: "ListStringNotInOptions",
			Parameter: provider.Parameter{
				Type:     "list(string)",
				Default:  ptr(`["red"]`),
				Option:   opts("red", "blue", "green"),
				FormType: provider.ParameterFormTypeMultiSelect,
			},
			Value:       `["red", "yellow", "black"]`,
			ExpectError: regexp.MustCompile("is not a valid option, values \"yellow, black\" are missing from the options"),
		},
		{
			Name: "InvalidMiniumum",
			Parameter: provider.Parameter{
				Type:    "number",
				Default: ptr("5"),
				Validation: []provider.Validation{{
					Min:   10,
					Error: "must be greater than 10",
				}},
			},
			ExpectError: regexp.MustCompile("must be greater than 10"),
		},
	} {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			value := &tc.Value
			_, diags := tc.Parameter.ValidateInput(value, nil)
			if tc.ExpectError != nil {
				require.True(t, diags.HasError())
				errMsg := fmt.Sprintf("%+v", diags[0]) // close enough
				require.Truef(t, tc.ExpectError.MatchString(errMsg), "got: %s", errMsg)
			} else {
				if !assert.False(t, diags.HasError()) {
					t.Logf("got: %+v", diags[0])
				}
			}
		})
	}
}

// TestParameterValidationEnforcement tests various parameter states and the
// validation enforcement that should be applied to them. The table is described
// by a markdown table. This is done so that the test cases can be more easily
// edited and read.
//
// Copy and paste the table to https://www.tablesgenerator.com/markdown_tables for easier editing
//
//nolint:paralleltest,tparallel // Parameters load values from env vars
func TestParameterValidationEnforcement(t *testing.T) {
	// Some interesting observations:
	// - Validation logic does not apply to the value of 'options'
	//	- [NumDefInvOpt] So an invalid option can be present and selected, but would fail
	// - Validation logic does not apply to the default if a value is given
	//	- [NumIns/DefInv] So the default can be invalid if an input value is valid.
	//	  The value is therefore not really optional, but it is marked as such.
	table, err := os.ReadFile("testdata/parameter_table.md")
	require.NoError(t, err)

	type row struct {
		Name        string
		Types       []string
		InputValue  string
		Default     string
		Options     []string
		Validation  *provider.Validation
		OutputValue string
		Optional    bool
		CreateError *regexp.Regexp
		Previous    *string
	}

	rows := make([]row, 0)
	lines := strings.Split(string(table), "\n")
	validMinMax := regexp.MustCompile("^[0-9]*-[0-9]*$")
	for _, line := range lines[2:] {
		columns := strings.Split(line, "|")
		columns = columns[1 : len(columns)-1]
		for i := range columns {
			// Trim the whitespace from all columns
			columns[i] = strings.TrimSpace(columns[i])
		}

		if columns[0] == "" {
			continue // Skip rows with empty names
		}

		cname, ctype, cprev, cinput, cdefault, coptions, cvalidation, _, coutput, coptional, cerr :=
			columns[0], columns[1], columns[2], columns[3], columns[4], columns[5], columns[6], columns[7], columns[8], columns[9], columns[10]

		optional, err := strconv.ParseBool(coptional)
		if coptional != "" {
			// Value does not matter if not specified
			require.NoError(t, err)
		}

		var rerr *regexp.Regexp
		if cerr != "" {
			rerr, err = regexp.Compile(cerr)
			if err != nil {
				t.Fatalf("failed to parse error column %q: %v", cerr, err)
			}
		}

		var options []string
		if coptions != "" {
			options = strings.Split(coptions, ",")
		}

		var validation *provider.Validation
		if cvalidation != "" {
			switch {
			case cvalidation == provider.ValidationMonotonicIncreasing || cvalidation == provider.ValidationMonotonicDecreasing:
				validation = &provider.Validation{
					MinDisabled: true,
					MaxDisabled: true,
					Monotonic:   cvalidation,
					Error:       "monotonicity",
				}
			case validMinMax.MatchString(cvalidation):
				// Min-Max validation should look like:
				//	1-10    :: min=1, max=10
				//	-10     :: max=10
				//	1-      :: min=1
				parts := strings.Split(cvalidation, "-")
				min, _ := strconv.ParseInt(parts[0], 10, 64)
				max, _ := strconv.ParseInt(parts[1], 10, 64)
				validation = &provider.Validation{
					Min:         int(min),
					MinDisabled: parts[0] == "",
					Max:         int(max),
					MaxDisabled: parts[1] == "",
					Monotonic:   "",
					Regex:       "",
					Error:       "{min} < {value} < {max}",
				}
			default:
				validation = &provider.Validation{
					Min:         0,
					MinDisabled: true,
					Max:         0,
					MaxDisabled: true,
					Monotonic:   "",
					Regex:       cvalidation,
					Error:       "regex error",
				}
			}
		}

		var prev *string
		if cprev != "" {
			prev = ptr(cprev)
			if cprev == `""` {
				prev = ptr("")
			}
		}
		rows = append(rows, row{
			Name:        cname,
			Types:       strings.Split(ctype, ","),
			InputValue:  cinput,
			Default:     cdefault,
			Options:     options,
			Validation:  validation,
			OutputValue: coutput,
			Optional:    optional,
			CreateError: rerr,
			Previous:    prev,
		})
	}

	stringLiteral := func(s string) string {
		if s == "" {
			return `""`
		}
		return fmt.Sprintf("%q", s)
	}

	for rowIndex, row := range rows {
		for _, rt := range row.Types {
			//nolint:paralleltest,tparallel // Parameters load values from env vars
			t.Run(fmt.Sprintf("%d|%s:%s", rowIndex, row.Name, rt), func(t *testing.T) {
				if row.InputValue != "" {
					t.Setenv(provider.ParameterEnvironmentVariable("parameter"), row.InputValue)
				}
				if row.Previous != nil {
					t.Setenv(provider.ParameterEnvironmentVariablePrevious("parameter"), *row.Previous)
				}

				if row.CreateError != nil && row.OutputValue != "" {
					t.Errorf("output value %q should not be set if both errors are set", row.OutputValue)
				}

				var cfg strings.Builder
				cfg.WriteString("data \"coder_parameter\" \"parameter\" {\n")
				cfg.WriteString("\tname = \"parameter\"\n")
				if rt == "multi-select" || rt == "tag-select" {
					cfg.WriteString(fmt.Sprintf("\ttype = \"%s\"\n", "list(string)"))
					cfg.WriteString(fmt.Sprintf("\tform_type = \"%s\"\n", rt))
				} else {
					cfg.WriteString(fmt.Sprintf("\ttype = \"%s\"\n", rt))
				}
				if row.Default != "" {
					cfg.WriteString(fmt.Sprintf("\tdefault = %s\n", stringLiteral(row.Default)))
				}

				for _, opt := range row.Options {
					cfg.WriteString("\toption {\n")
					cfg.WriteString(fmt.Sprintf("\t\tname = %s\n", stringLiteral(opt)))
					cfg.WriteString(fmt.Sprintf("\t\tvalue = %s\n", stringLiteral(opt)))
					cfg.WriteString("\t}\n")
				}

				if row.Validation != nil {
					cfg.WriteString("\tvalidation {\n")
					if !row.Validation.MinDisabled {
						cfg.WriteString(fmt.Sprintf("\t\tmin = %d\n", row.Validation.Min))
					}
					if !row.Validation.MaxDisabled {
						cfg.WriteString(fmt.Sprintf("\t\tmax = %d\n", row.Validation.Max))
					}
					if row.Validation.Monotonic != "" {
						cfg.WriteString(fmt.Sprintf("\t\tmonotonic = \"%s\"\n", row.Validation.Monotonic))
					}
					if row.Validation.Regex != "" {
						cfg.WriteString(fmt.Sprintf("\t\tregex = %q\n", row.Validation.Regex))
					}
					cfg.WriteString(fmt.Sprintf("\t\terror = %q\n", row.Validation.Error))
					cfg.WriteString("\t}\n")
				}

				cfg.WriteString("}\n")
				resource.Test(t, resource.TestCase{
					ProviderFactories: coderFactory(),
					IsUnitTest:        true,
					Steps: []resource.TestStep{{
						Config:      cfg.String(),
						ExpectError: row.CreateError,
						Check: func(state *terraform.State) error {
							require.Len(t, state.Modules, 1)
							require.Len(t, state.Modules[0].Resources, 1)
							param := state.Modules[0].Resources["data.coder_parameter.parameter"]
							require.NotNil(t, param)

							if row.Default == "" {
								_, ok := param.Primary.Attributes["default"]
								require.False(t, ok, "default should not be set")
							} else {
								require.Equal(t, strings.Trim(row.Default, `"`), param.Primary.Attributes["default"])
							}

							if row.OutputValue == "" {
								_, ok := param.Primary.Attributes["value"]
								require.False(t, ok, "output value should not be set")
							} else {
								require.Equal(t, strings.Trim(row.OutputValue, `"`), param.Primary.Attributes["value"])
							}

							for key, expected := range map[string]string{
								"optional": strconv.FormatBool(row.Optional),
							} {
								require.Equal(t, expected, param.Primary.Attributes[key], "optional")
							}

							return nil
						},
					}},
				})
			})
		}
	}
}

func TestValueValidatesType(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		Name                     string
		Type                     provider.OptionType
		Value                    string
		Previous                 *string
		Regex                    string
		RegexError               string
		Min                      int
		Max                      int
		MinDisabled, MaxDisabled bool
		Monotonic                string
		Error                    *regexp.Regexp
	}{{
		Name:        "StringWithMin",
		Type:        "string",
		Min:         1,
		MaxDisabled: true,
		Error:       regexp.MustCompile("cannot be specified"),
	}, {
		Name:        "StringWithMax",
		Type:        "string",
		Max:         1,
		MinDisabled: true,
		Error:       regexp.MustCompile("cannot be specified"),
	}, {
		Name:  "NonStringWithRegex",
		Type:  "number",
		Regex: "banana",
		Error: regexp.MustCompile("a regex cannot be specified"),
	}, {
		Name:        "Bool",
		Type:        "bool",
		Value:       "true",
		MinDisabled: true,
		MaxDisabled: true,
	}, {
		Name:  "InvalidNumber",
		Type:  "number",
		Value: "hi",
		Error: regexp.MustCompile("is not a number"),
	}, {
		Name:        "NumberBelowMin",
		Type:        "number",
		Value:       "0",
		Min:         1,
		MaxDisabled: true,
		Error:       regexp.MustCompile("is less than the minimum 1"),
	}, {
		Name:        "NumberAboveMax",
		Type:        "number",
		Value:       "2",
		Max:         1,
		MinDisabled: true,
		Error:       regexp.MustCompile("is more than the maximum 1"),
	}, {
		Name:        "InvalidBool",
		Type:        "bool",
		Value:       "cat",
		MinDisabled: true,
		MaxDisabled: true,
		Error:       regexp.MustCompile("boolean value can be either"),
	}, {
		Name:        "BadStringWithRegex",
		Type:        "string",
		Regex:       "banana",
		RegexError:  "bad fruit",
		Value:       "apple",
		MinDisabled: true,
		MaxDisabled: true,
		Error:       regexp.MustCompile(`bad fruit`),
	}, {
		Name:      "InvalidMonotonicity",
		Type:      "number",
		Value:     "1",
		Min:       0,
		Max:       2,
		Monotonic: "foobar",
		Error:     regexp.MustCompile(`number monotonicity can be either "increasing" or "decreasing"`),
	}, {
		Name:      "IncreasingMonotonicity",
		Type:      "number",
		Value:     "1",
		Min:       0,
		Max:       2,
		Monotonic: "increasing",
	}, {
		Name:      "DecreasingMonotonicity",
		Type:      "number",
		Value:     "1",
		Min:       0,
		Max:       2,
		Monotonic: "decreasing",
	}, {
		Name:        "IncreasingMonotonicityEqual",
		Type:        "number",
		Previous:    ptr("1"),
		Value:       "1",
		Monotonic:   "increasing",
		MinDisabled: true,
		MaxDisabled: true,
	}, {
		Name:        "DecreasingMonotonicityEqual",
		Type:        "number",
		Value:       "1",
		Previous:    ptr("1"),
		Monotonic:   "decreasing",
		MinDisabled: true,
		MaxDisabled: true,
	}, {
		Name:        "IncreasingMonotonicityGreater",
		Type:        "number",
		Previous:    ptr("0"),
		Value:       "1",
		Monotonic:   "increasing",
		MinDisabled: true,
		MaxDisabled: true,
	}, {
		Name:        "DecreasingMonotonicityGreater",
		Type:        "number",
		Value:       "1",
		Previous:    ptr("0"),
		Monotonic:   "decreasing",
		MinDisabled: true,
		MaxDisabled: true,
		Error:       regexp.MustCompile("must be equal or"),
	}, {
		Name:        "IncreasingMonotonicityLesser",
		Type:        "number",
		Previous:    ptr("2"),
		Value:       "1",
		Monotonic:   "increasing",
		MinDisabled: true,
		MaxDisabled: true,
		Error:       regexp.MustCompile("must be equal or"),
	}, {
		Name:        "DecreasingMonotonicityLesser",
		Type:        "number",
		Value:       "1",
		Previous:    ptr("2"),
		Monotonic:   "decreasing",
		MinDisabled: true,
		MaxDisabled: true,
	}, {
		Name:        "ValidListOfStrings",
		Type:        "list(string)",
		Value:       `["first","second","third"]`,
		MinDisabled: true,
		MaxDisabled: true,
	}, {
		Name:        "InvalidListOfStrings",
		Type:        "list(string)",
		Value:       `["first","second","third"`,
		MinDisabled: true,
		MaxDisabled: true,
		Error:       regexp.MustCompile("is not valid list of strings"),
	}, {
		Name:        "EmptyListOfStrings",
		Type:        "list(string)",
		Value:       `[]`,
		MinDisabled: true,
		MaxDisabled: true,
	}, {
		Name:        "ValidListOfStrings",
		Type:        "list(string)",
		Value:       `["first","second","third"]`,
		MinDisabled: true,
		MaxDisabled: true,
	}, {
		Name:        "InvalidListOfStrings",
		Type:        "list(string)",
		Value:       `["first","second","third"`,
		MinDisabled: true,
		MaxDisabled: true,
		Error:       regexp.MustCompile("is not valid list of strings"),
	}, {
		Name:        "EmptyListOfStrings",
		Type:        "list(string)",
		Value:       `[]`,
		MinDisabled: true,
		MaxDisabled: true,
	}} {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			v := &provider.Validation{
				Min:         tc.Min,
				MinDisabled: tc.MinDisabled,
				Max:         tc.Max,
				MaxDisabled: tc.MaxDisabled,
				Monotonic:   tc.Monotonic,
				Regex:       tc.Regex,
				Error:       tc.RegexError,
			}
			err := v.Valid(tc.Type, tc.Value, tc.Previous)
			if tc.Error != nil {
				require.Error(t, err)
				require.True(t, tc.Error.MatchString(err.Error()), "got: %s", err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParameterWithManyOptions(t *testing.T) {
	t.Parallel()

	const maxItemsInTest = 1024

	var options strings.Builder
	for i := 0; i < maxItemsInTest; i++ {
		_, _ = options.WriteString(fmt.Sprintf(`option {
					name = "%d"
					value = "%d"
				}
`, i, i))
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
		Steps: []resource.TestStep{{
			Config: fmt.Sprintf(`data "coder_parameter" "region" {
				name = "Region"
				type = "string"
				%s
			}`, options.String()),
			Check: func(state *terraform.State) error {
				require.Len(t, state.Modules, 1)
				require.Len(t, state.Modules[0].Resources, 1)
				param := state.Modules[0].Resources["data.coder_parameter.region"]

				for i := 0; i < maxItemsInTest; i++ {
					name, _ := param.Primary.Attributes[fmt.Sprintf("option.%d.name", i)]
					value, _ := param.Primary.Attributes[fmt.Sprintf("option.%d.value", i)]
					require.Equal(t, fmt.Sprintf("%d", i), name)
					require.Equal(t, fmt.Sprintf("%d", i), value)
				}
				return nil
			},
		}},
	})
}

func ptr[T any](v T) *T {
	return &v
}
