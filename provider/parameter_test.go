package provider_test

import (
	"regexp"
	"testing"

	"github.com/coder/terraform-provider-coder/provider"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
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
						description = <<-EOT
							# Select the machine image
							See the [registry](https://container.registry.blah/namespace) for options.
							EOT
						mutable = true
						icon = "/icon/region.svg"
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
				"ephemeral":            "true",
			} {
				require.Equal(t, value, attrs[key])
			}
		},
	}, {
		Name: "ValidationWithOptions",
		Config: `
					data "coder_parameter" "region" {
						name = "Region"
						type = "number"
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
		ExpectError: regexp.MustCompile("conflicts with option"),
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
		ExpectError: regexp.MustCompile("cannot have the same name"),
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
		ExpectError: regexp.MustCompile("cannot have the same value"),
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
		Name: "LegacyVariable",
		Config: `
		variable "old_region" {
		  type = string
		  default = "fake-region" # for testing purposes, no need to set via env TF_...
		}

		data "coder_parameter" "region" {
			name = "Region"
			type = "string"
			default = "will-be-ignored"
			legacy_variable_name = "old_region"
			legacy_variable = var.old_region
		}`,
		Check: func(state *terraform.ResourceState) {
			for key, expected := range map[string]string{
				"name":                 "Region",
				"type":                 "string",
				"default":              "fake-region",
				"legacy_variable_name": "old_region",
				"legacy_variable":      "fake-region",
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
		Name: "ListOfStringsButMigrated",
		Config: `
		variable "old_region" {
			type = list(string)
			default = ["us-west-1a"] # for testing purposes, no need to set via env TF_...
		}

		data "coder_parameter" "region" {
			name = "Region"
			type = "list(string)"
			default = "[\"us-east-1\", \"eu-west-1\", \"ap-northeast-1\"]"
			legacy_variable_name = "old_region"
			legacy_variable = jsonencode(var.old_region)
		}`,
		Check: func(state *terraform.ResourceState) {
			for key, expected := range map[string]string{
				"name":    "Region",
				"type":    "list(string)",
				"default": `["us-west-1a"]`,
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
				mutable = false
				ephemeral = true
			}
			`,
		ExpectError: regexp.MustCompile("parameter can't be immutable and ephemeral"),
	}} {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			resource.Test(t, resource.TestCase{
				Providers: map[string]*schema.Provider{
					"coder": provider.New(),
				},
				IsUnitTest: true,
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

func TestValueValidatesType(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		Name,
		Type,
		Value,
		Regex,
		RegexError string
		Min,
		Max int
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
			err := v.Valid(tc.Type, tc.Value)
			if tc.Error != nil {
				require.Error(t, err)
				require.True(t, tc.Error.MatchString(err.Error()), "got: %s", err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
