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
	formType      provider.ParameterFormType
	optionType    provider.OptionType
	defValue      string
	options       bool
	customOptions []string
}

func (c formTypeCheck) String() string {
	return fmt.Sprintf("%s_%s_%t", c.formType, c.optionType, c.options)
}

func TestValidateFormType(t *testing.T) {
	t.Parallel()

	// formTypesChecked keeps track of all checks run. It will be used to
	// ensure all combinations of form_type and option_type are tested.
	// All untested options are assumed to throw an error.
	formTypesChecked := make(map[string]struct{})

	expectType := func(expected provider.ParameterFormType, opts formTypeCheck) formTypeTestCase {
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

	// obvious just assumes the FormType in the check is the expected
	// FormType. Using `expectType` these fields can differ
	obvious := func(opts formTypeCheck) formTypeTestCase {
		return expectType(opts.formType, opts)
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
		// All default behaviors. Essentially legacy behavior.
		//	String
		expectType(provider.ParameterFormTypeRadio, formTypeCheck{
			options:    true,
			optionType: provider.OptionTypeString,
		}),
		expectType(provider.ParameterFormTypeInput, formTypeCheck{
			options:    false,
			optionType: provider.OptionTypeString,
		}),
		//	Number
		expectType(provider.ParameterFormTypeRadio, formTypeCheck{
			options:    true,
			optionType: provider.OptionTypeNumber,
		}),
		expectType(provider.ParameterFormTypeInput, formTypeCheck{
			options:    false,
			optionType: provider.OptionTypeNumber,
		}),
		//	Boolean
		expectType(provider.ParameterFormTypeRadio, formTypeCheck{
			options:    true,
			optionType: provider.OptionTypeBoolean,
		}),
		expectType(provider.ParameterFormTypeCheckbox, formTypeCheck{
			options:    false,
			optionType: provider.OptionTypeBoolean,
		}),
		//	List(string)
		expectType(provider.ParameterFormTypeRadio, formTypeCheck{
			options:    true,
			optionType: provider.OptionTypeListString,
		}),
		expectType(provider.ParameterFormTypeTagSelect, formTypeCheck{
			options:    false,
			optionType: provider.OptionTypeListString,
		}),

		// ---- New Behavior
		//	String
		obvious(formTypeCheck{
			options:    true,
			optionType: provider.OptionTypeString,
			formType:   provider.ParameterFormTypeDropdown,
		}),
		obvious(formTypeCheck{
			options:    true,
			optionType: provider.OptionTypeString,
			formType:   provider.ParameterFormTypeRadio,
		}),
		obvious(formTypeCheck{
			options:    false,
			optionType: provider.OptionTypeString,
			formType:   provider.ParameterFormTypeInput,
		}),
		obvious(formTypeCheck{
			options:    false,
			optionType: provider.OptionTypeString,
			formType:   provider.ParameterFormTypeTextArea,
		}),
		//	Number
		obvious(formTypeCheck{
			options:    true,
			optionType: provider.OptionTypeNumber,
			formType:   provider.ParameterFormTypeDropdown,
		}),
		obvious(formTypeCheck{
			options:    true,
			optionType: provider.OptionTypeNumber,
			formType:   provider.ParameterFormTypeRadio,
		}),
		obvious(formTypeCheck{
			options:    false,
			optionType: provider.OptionTypeNumber,
			formType:   provider.ParameterFormTypeInput,
		}),
		obvious(formTypeCheck{
			options:    false,
			optionType: provider.OptionTypeNumber,
			formType:   provider.ParameterFormTypeSlider,
		}),
		//	Boolean
		obvious(formTypeCheck{
			options:    true,
			optionType: provider.OptionTypeBoolean,
			formType:   provider.ParameterFormTypeRadio,
		}),
		obvious(formTypeCheck{
			options:    false,
			optionType: provider.OptionTypeBoolean,
			formType:   provider.ParameterFormTypeSwitch,
		}),
		obvious(formTypeCheck{
			options:    false,
			optionType: provider.OptionTypeBoolean,
			formType:   provider.ParameterFormTypeCheckbox,
		}),
		//	List(string)
		obvious(formTypeCheck{
			options:    true,
			optionType: provider.OptionTypeListString,
			formType:   provider.ParameterFormTypeRadio,
		}),
		obvious(formTypeCheck{
			options:       true,
			optionType:    provider.OptionTypeListString,
			formType:      provider.ParameterFormTypeMultiSelect,
			customOptions: []string{"red", "blue", "green"},
			defValue:      `["red", "blue"]`,
		}),
		obvious(formTypeCheck{
			options:    false,
			optionType: provider.OptionTypeListString,
			formType:   provider.ParameterFormTypeTagSelect,
		}),
	}

	t.Run("TabledTests", func(t *testing.T) {
		// TabledCases runs through all the manual test cases
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				if c.assert.Styling == "" {
					c.assert.Styling = "{}"
				}

				formTypeTest(t, c)
				if _, ok := formTypesChecked[c.config.String()]; ok {
					t.Log("Duplicated form type check, delete this extra test case")
					t.Fatalf("form type %q already checked", c.config.String())
				}
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

func ezconfig(paramName string, cfg formTypeCheck) (defaultValue string, tf string) {
	var body strings.Builder
	if cfg.defValue != "" {
		body.WriteString(fmt.Sprintf("default = %q\n", cfg.defValue))
	}
	if cfg.formType != "" {
		body.WriteString(fmt.Sprintf("form_type = %q\n", cfg.formType))
	}
	if cfg.optionType != "" {
		body.WriteString(fmt.Sprintf("type = %q\n", cfg.optionType))
	}

	options := cfg.customOptions
	if cfg.options && len(cfg.customOptions) == 0 {
		switch cfg.optionType {
		case provider.OptionTypeString:
			options = []string{"foo"}
			defaultValue = "foo"
		case provider.OptionTypeBoolean:
			options = []string{"true", "false"}
			defaultValue = "true"
		case provider.OptionTypeNumber:
			options = []string{"1"}
			defaultValue = "1"
		case provider.OptionTypeListString:
			options = []string{`["red", "blue"]`}
			defaultValue = `["red"]`
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

	if cfg.defValue == "" {
		cfg.defValue = defaultValue
	}
	return cfg.defValue, fmt.Sprintf(`
				provider "coder" {
				}
				data "coder_parameter" "%s" {
					name = "%s"
					%s
				}
		`, paramName, paramName, body.String())
}

func formTypeTest(t *testing.T, c formTypeTestCase) {
	t.Helper()
	const paramName = "test_param"

	def, tf := ezconfig(paramName, c.config)
	t.Logf("Terraform config:\n%s", tf)
	checkFn := func(state *terraform.State) error {
		require.Len(t, state.Modules, 1)
		require.Len(t, state.Modules[0].Resources, 1)

		key := strings.Join([]string{"data", "coder_parameter", paramName}, ".")
		param := state.Modules[0].Resources[key]

		assert.Equal(t, def, param.Primary.Attributes["default"], "default value")
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
				Config:      tf,
				Check:       checkFn,
				ExpectError: c.expectError,
			},
		},
	})
}
