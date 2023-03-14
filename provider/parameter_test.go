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
				name = "Region"
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
			}
			`,
		Check: func(state *terraform.ResourceState) {
			attrs := state.Primary.Attributes
			for key, value := range map[string]interface{}{
				"name":                 "Region",
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
		Monotonic string
		Error     *regexp.Regexp
	}{{
		Name:  "StringWithMin",
		Type:  "string",
		Min:   1,
		Error: regexp.MustCompile("cannot be specified"),
	}, {
		Name:  "StringWithMax",
		Type:  "string",
		Max:   1,
		Error: regexp.MustCompile("cannot be specified"),
	}, {
		Name:  "NonStringWithRegex",
		Type:  "number",
		Regex: "banana",
		Error: regexp.MustCompile("a regex cannot be specified"),
	}, {
		Name:  "Bool",
		Type:  "bool",
		Value: "true",
	}, {
		Name:  "InvalidNumber",
		Type:  "number",
		Value: "hi",
		Error: regexp.MustCompile("is not a number"),
	}, {
		Name:  "NumberBelowMin",
		Type:  "number",
		Value: "0",
		Min:   1,
		Error: regexp.MustCompile("is less than the minimum"),
	}, {
		Name:  "NumberAboveMax",
		Type:  "number",
		Value: "1",
		Max:   0,
		Error: regexp.MustCompile("is more than the maximum"),
	}, {
		Name:  "InvalidBool",
		Type:  "bool",
		Value: "cat",
		Error: regexp.MustCompile("boolean value can be either"),
	}, {
		Name:       "BadStringWithRegex",
		Type:       "string",
		Regex:      "banana",
		RegexError: "bad fruit",
		Value:      "apple",
		Error:      regexp.MustCompile(`bad fruit`),
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
	}} {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			v := &provider.Validation{
				Min:       tc.Min,
				Max:       tc.Max,
				Monotonic: tc.Monotonic,
				Regex:     tc.Regex,
				Error:     tc.RegexError,
			}
			err := v.Valid(tc.Type, tc.Value)
			if tc.Error != nil {
				require.Error(t, err)
				require.True(t, tc.Error.MatchString(err.Error()), "got: %s", err.Error())
			}
		})
	}
}
