package provider

import (
	"slices"

	"golang.org/x/xerrors"
)

// OptionType is a type of option that can be used in the 'type' argument of
// a parameter.
type OptionType string

const (
	OptionTypeString     OptionType = "string"
	OptionTypeNumber     OptionType = "number"
	OptionTypeBoolean    OptionType = "bool"
	OptionTypeListString OptionType = "list(string)"
)

func OptionTypes() []OptionType {
	return []OptionType{
		OptionTypeString,
		OptionTypeNumber,
		OptionTypeBoolean,
		OptionTypeListString,
	}
}

type ParameterFormType string

const (
	ParameterFormTypeDefault     ParameterFormType = ""
	ParameterFormTypeRadio       ParameterFormType = "radio"
	ParameterFormTypeSlider      ParameterFormType = "slider"
	ParameterFormTypeInput       ParameterFormType = "input"
	ParameterFormTypeDropdown    ParameterFormType = "dropdown"
	ParameterFormTypeCheckbox    ParameterFormType = "checkbox"
	ParameterFormTypeSwitch      ParameterFormType = "switch"
	ParameterFormTypeMultiSelect ParameterFormType = "multi-select"
	ParameterFormTypeTagSelect   ParameterFormType = "tag-select"
	ParameterFormTypeTextArea    ParameterFormType = "textarea"
	ParameterFormTypeError       ParameterFormType = "error"
)

// ParameterFormTypes should be kept in sync with the enum list above.
func ParameterFormTypes() []ParameterFormType {
	return []ParameterFormType{
		ParameterFormTypeDefault,
		ParameterFormTypeRadio,
		ParameterFormTypeSlider,
		ParameterFormTypeInput,
		ParameterFormTypeDropdown,
		ParameterFormTypeCheckbox,
		ParameterFormTypeSwitch,
		ParameterFormTypeMultiSelect,
		ParameterFormTypeTagSelect,
		ParameterFormTypeTextArea,
		ParameterFormTypeError,
	}
}

// formTypeTruthTable is a map of [`type`][`optionCount` > 0] to `form_type`.
// The first value in the slice is the default value assuming `form_type` is
// not specified.
//
// The boolean key indicates whether the `options` field is specified.
// | Type              | Options | Specified Form Type | form_type      | Notes                          |
// |-------------------|---------|---------------------|----------------|--------------------------------|
// | `string` `number` | Y       |                     | `radio`        |                                |
// | `string` `number` | Y       | `dropdown`          | `dropdown`     |                                |
// | `string` `number` | N       |                     | `input`        |                                |
// | `string`          | N       | 'textarea'          | `textarea`     |                                |
// | `number`          | N       | 'slider'            | `slider`       | min/max validation             |
// | `bool`            | Y       |                     | `radio`        |                                |
// | `bool`            | N       |                     | `checkbox`     |                                |
// | `bool`            | N       | `switch`            | `switch`       |                                |
// | `list(string)`    | Y       |                     | `radio`        |                                |
// | `list(string)`    | N       |                     | `tag-select`   |                                |
// | `list(string)`    | Y       | `multi-select`      | `multi-select` | Option values will be `string` |
var formTypeTruthTable = map[OptionType]map[bool][]ParameterFormType{
	OptionTypeString: {
		true:  {ParameterFormTypeRadio, ParameterFormTypeDropdown},
		false: {ParameterFormTypeInput, ParameterFormTypeTextArea},
	},
	OptionTypeNumber: {
		true:  {ParameterFormTypeRadio, ParameterFormTypeDropdown},
		false: {ParameterFormTypeInput, ParameterFormTypeSlider},
	},
	OptionTypeBoolean: {
		true:  {ParameterFormTypeRadio},
		false: {ParameterFormTypeCheckbox, ParameterFormTypeSwitch},
	},
	OptionTypeListString: {
		true:  {ParameterFormTypeRadio, ParameterFormTypeMultiSelect},
		false: {ParameterFormTypeTagSelect},
	},
}

// ValidateFormType handles the truth table for the valid set of `type` and
// `form_type` options.
// The OptionType is also returned because it is possible the 'type' of the
// 'value' & 'default' fields is different from the 'type' of the options.
// The use case is when using multi-select. The options are 'string' and the
// value is 'list(string)'.
func ValidateFormType(paramType OptionType, optionCount int, specifiedFormType ParameterFormType) (OptionType, ParameterFormType, error) {
	allowed, ok := formTypeTruthTable[paramType][optionCount > 0]
	if !ok || len(allowed) == 0 {
		return paramType, specifiedFormType, xerrors.Errorf("value type %q is not supported for 'form_types'", paramType)
	}

	if specifiedFormType == ParameterFormTypeDefault {
		// handle the default case
		specifiedFormType = allowed[0]
	}

	if !slices.Contains(allowed, specifiedFormType) {
		return paramType, specifiedFormType, xerrors.Errorf("value type %q is not supported for 'form_types'", specifiedFormType)
	}

	// Special case
	if paramType == OptionTypeListString && specifiedFormType == ParameterFormTypeMultiSelect {
		return OptionTypeListString, ParameterFormTypeMultiSelect, nil
	}

	return paramType, specifiedFormType, nil
}

func toStrings[A ~string](l []A) []string {
	var r []string
	for _, v := range l {
		r = append(r, string(v))
	}
	return r
}
