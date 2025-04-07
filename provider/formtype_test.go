package provider_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/coder/terraform-provider-coder/v2/provider"
)

type formTypeTestCase struct {
	name        string
	config      formTypeCheck
	assert      paramAssert
	expectError *regexp.Regexp
}

type paramAssert struct {
	FormType provider.ParameterFormType
	Type     provider.OptionType
	Styling  string
}

type formTypeCheck struct {
	formType   provider.ParameterFormType
	optionType provider.OptionType
	options    bool
}

func (c formTypeCheck) String() string {
	return fmt.Sprintf("%s_%s_%t", c.formType, c.optionType, c.options)
}

func TestValidateFormType(t *testing.T) {
	t.Parallel()

	//formTypesChecked := make(map[provider.ParameterFormType]map[provider.OptionType]map[bool]struct{})
	formTypesChecked := make(map[string]struct{})

	obvious := func(expected provider.ParameterFormType, opts formTypeCheck) formTypeTestCase {
		ftname := opts.formType
		if ftname == "" {
			ftname = "default"
		}
		return formTypeTestCase{
			name: fmt.Sprintf("%s_%s_%t",
				ftname,
				opts.optionType,
				opts.options,
			),
			config: opts,
			assert: paramAssert{
				FormType: expected,
				Type:     opts.optionType,
				Styling:  "",
			},
			expectError: nil,
		}
	}

	cases := []formTypeTestCase{
		{
			// When nothing is specified
			name:   "defaults",
			config: formTypeCheck{},
			assert: paramAssert{
				FormType: provider.ParameterFormTypeInput,
				Type:     provider.OptionTypeString,
				Styling:  "",
			},
		},
		// String
		obvious(provider.ParameterFormTypeRadio, formTypeCheck{
			options:    true,
			optionType: provider.OptionTypeString,
		}),
		obvious(provider.ParameterFormTypeRadio, formTypeCheck{
			options:    true,
			optionType: provider.OptionTypeString,
			formType:   provider.ParameterFormTypeRadio,
		}),
		obvious(provider.ParameterFormTypeDropdown, formTypeCheck{
			options:    true,
			optionType: provider.OptionTypeString,
			formType:   provider.ParameterFormTypeDropdown,
		}),
		obvious(provider.ParameterFormTypeInput, formTypeCheck{
			options:    false,
			optionType: provider.OptionTypeString,
		}),
		obvious(provider.ParameterFormTypeTextArea, formTypeCheck{
			options:    false,
			optionType: provider.OptionTypeString,
			formType:   provider.ParameterFormTypeTextArea,
		}),
		// Numbers
		obvious(provider.ParameterFormTypeRadio, formTypeCheck{
			options:    true,
			optionType: provider.OptionTypeNumber,
		}),
		obvious(provider.ParameterFormTypeRadio, formTypeCheck{
			options:    true,
			optionType: provider.OptionTypeNumber,
			formType:   provider.ParameterFormTypeRadio,
		}),
		obvious(provider.ParameterFormTypeDropdown, formTypeCheck{
			options:    true,
			optionType: provider.OptionTypeNumber,
			formType:   provider.ParameterFormTypeDropdown,
		}),
		obvious(provider.ParameterFormTypeInput, formTypeCheck{
			options:    false,
			optionType: provider.OptionTypeNumber,
		}),
		obvious(provider.ParameterFormTypeSlider, formTypeCheck{
			options:    false,
			optionType: provider.OptionTypeNumber,
			formType:   provider.ParameterFormTypeSlider,
		}),
		// booleans
		obvious(provider.ParameterFormTypeRadio, formTypeCheck{
			options:    true,
			optionType: provider.OptionTypeBoolean,
		}),
		obvious(provider.ParameterFormTypeCheckbox, formTypeCheck{
			options:    false,
			optionType: provider.OptionTypeBoolean,
		}),
		obvious(provider.ParameterFormTypeCheckbox, formTypeCheck{
			options:    false,
			optionType: provider.OptionTypeBoolean,
			formType:   provider.ParameterFormTypeCheckbox,
		}),
		obvious(provider.ParameterFormTypeSwitch, formTypeCheck{
			options:    false,
			optionType: provider.OptionTypeBoolean,
			formType:   provider.ParameterFormTypeSwitch,
		}),
	}

	// TabledCases runs through all the manual test cases
	t.Run("TabledCases", func(t *testing.T) {
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				if c.assert.Styling == "" {
					c.assert.Styling = "{}"
				}

				formTypeTest(t, c)
				formTypesChecked[c.config.String()] = struct{}{}
			})
		}
	})

	// AssumeErrorCases assumes any uncovered test will return an error.
	// This ensures all valid test case paths are covered.
	t.Run("AssumeErrorCases", func(t *testing.T) {
		// requiredChecks loops through all possible form_type and option_type
		// combinations.
		requiredChecks := make([]formTypeCheck, 0)
		//requiredChecks := make(map[provider.ParameterFormType][]provider.OptionType)
		for _, ft := range append(provider.ParameterFormTypes(), "") {
			//requiredChecks[ft] = make([]provider.OptionType, 0)
			for _, ot := range provider.OptionTypes() {
				requiredChecks = append(requiredChecks, formTypeCheck{
					formType:   ft,
					optionType: ot,
					options:    false,
				})
				requiredChecks = append(requiredChecks, formTypeCheck{
					formType:   ft,
					optionType: ot,
					options:    true,
				})
			}
		}

		for _, check := range requiredChecks {
			_, alreadyChecked := formTypesChecked[check.String()]
			if alreadyChecked {
				continue
			}

			ftName := check.formType
			if ftName == "" {
				ftName = "default"
			}
			fc := formTypeTestCase{
				name: fmt.Sprintf("%s_%s_%t",
					ftName,
					check.optionType,
					check.options,
				),
				config:      check,
				assert:      paramAssert{},
				expectError: regexp.MustCompile("is not supported"),
			}

			t.Run(fc.name, func(t *testing.T) {
				t.Parallel()

				tcText := fmt.Sprintf(`
					obvious(%s, ezconfigOpts{
						Options:    %t,
						OptionType: %q,
						FormType:   %q,
					}),
				`, "<expected_form_type>", check.options, check.optionType, check.formType)
				t.Logf("To construct this test case:\n%s", tcText)
				formTypeTest(t, fc)
			})

		}
	})
}

func ezconfig(paramName string, cfg formTypeCheck) string {
	var body strings.Builder
	//if cfg.Default != "" {
	//	body.WriteString(fmt.Sprintf("default = %q\n", cfg.Default))
	//}
	if cfg.formType != "" {
		body.WriteString(fmt.Sprintf("form_type = %q\n", cfg.formType))
	}
	if cfg.optionType != "" {
		body.WriteString(fmt.Sprintf("type = %q\n", cfg.optionType))
	}

	var options []string
	if cfg.options {
		switch cfg.optionType {
		case provider.OptionTypeString:
			options = []string{"foo"}
		case provider.OptionTypeBoolean:
			options = []string{"true", "false"}
		case provider.OptionTypeNumber:
			options = []string{"1"}
		case provider.OptionTypeListString:
			options = []string{`["red", "blue"]`}
		default:
			panic(fmt.Sprintf("unknown option type %q when generating options", cfg.optionType))
		}
	}

	for i, opt := range options {
		body.WriteString("option {\n")
		body.WriteString(fmt.Sprintf("name = \"val_%d\"\n", i))
		body.WriteString(fmt.Sprintf("value = %q\n", opt))
		body.WriteString("}\n")
	}

	return coderParamHCL(paramName, body.String())
}

func coderParamHCL(paramName string, body string) string {
	return fmt.Sprintf(`
				provider "coder" {
				}
				data "coder_parameter" "%s" {
					name = "%s"
					%s
				}
		`, paramName, paramName, body)
}

func formTypeTest(t *testing.T, c formTypeTestCase) {
	const paramName = "test_param"

	checkFn := func(state *terraform.State) error {
		require.Len(t, state.Modules, 1)
		require.Len(t, state.Modules[0].Resources, 1)

		key := strings.Join([]string{"data", "coder_parameter", paramName}, ".")
		param := state.Modules[0].Resources[key]

		//assert.Equal(t, c.assert.Default, param.Primary.Attributes["default"], "default value")
		assert.Equal(t, string(c.assert.FormType), param.Primary.Attributes["form_type"], "form_type")
		assert.Equal(t, string(c.assert.Type), param.Primary.Attributes["type"], "type")
		assert.JSONEq(t, c.assert.Styling, param.Primary.Attributes["styling"], "styling")

		//ft := provider.ParameterFormType(param.Primary.Attributes["form_type"])
		//ot := provider.OptionType(param.Primary.Attributes["type"])

		// Option blocks are not stored in a very friendly format
		// here.
		//options := param.Primary.Attributes["option.0.name"] != ""

		return nil
	}
	if c.expectError != nil {
		checkFn = nil
	}

	resource.Test(t, resource.TestCase{
		IsUnitTest:        true,
		ProviderFactories: coderFactory(),
		Steps: []resource.TestStep{
			{
				Config:      ezconfig(paramName, c.config),
				Check:       checkFn,
				ExpectError: c.expectError,
			},
		},
	})
}
