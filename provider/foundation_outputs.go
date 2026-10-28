package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/coder/terraform-provider-coder/v2/provider/helpers"
)

type foundationOutputMetadata struct {
	Type      json.RawMessage `json:"type"`
	Sensitive *bool           `json:"sensitive"`
}

// outputJSONObject rejects ambiguous duplicate keys as well as non-object JSON.
func outputJSONObject(data []byte) (map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return nil, errors.New("expected a JSON object")
	}
	object := make(map[string]json.RawMessage)
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		name, ok := key.(string)
		if !ok {
			return nil, errors.New("expected a JSON object key")
		}
		if _, exists := object[name]; exists {
			return nil, errors.New("duplicate JSON object key")
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
		object[name] = value
	}
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, errors.New("unexpected trailing JSON")
	}
	return object, nil
}

func parseFoundationOutputMetadata() (map[string]foundationOutputMetadata, error) {
	entries, err := outputJSONObject([]byte(helpers.OptionalEnvOrDefault("CODER_FOUNDATION_OUTPUT_METADATA", "{}")))
	if err != nil {
		return nil, errors.New("Foundation output metadata in CODER_FOUNDATION_OUTPUT_METADATA must be a JSON object")
	}
	metadata := make(map[string]foundationOutputMetadata, len(entries))
	for name, entry := range entries {
		fields, err := outputJSONObject(entry)
		var sensitive *bool
		if err != nil || json.Unmarshal(fields["sensitive"], &sensitive) != nil || sensitive == nil {
			return nil, errors.New("Foundation output metadata must include a Terraform JSON type and a boolean sensitive flag for every output")
		}
		if _, err := ctyjson.UnmarshalType(fields["type"]); err != nil {
			return nil, errors.New("Foundation output metadata must include a Terraform JSON type and a boolean sensitive flag for every output")
		}
		metadata[name] = foundationOutputMetadata{Type: fields["type"], Sensitive: sensitive}
	}
	return metadata, nil
}

func foundationOutputsSensitive() bool {
	// The legacy data source exposes a combined dynamic object. Protocol 5
	// cannot mark individual keys, so protect the whole attribute if needed.
	metadata, err := parseFoundationOutputMetadata()
	if err != nil {
		return true
	}
	for _, output := range metadata {
		if *output.Sensitive {
			return true
		}
	}
	return false
}

// foundationOutputSnapshot validates the entire pinned snapshot before any
// values can be published, including values destined for the other bucket.
func foundationOutputSnapshot(allowLegacyInference bool) (tftypes.Value, map[string]foundationOutputMetadata, error) {
	outputsJSON := []byte(helpers.OptionalEnvOrDefault("CODER_TEMPLATE_OUTPUTS", "{}"))
	outputs, err := outputJSONObject(outputsJSON)
	if err != nil {
		return tftypes.Value{}, nil, errors.New("Shared resource outputs in CODER_TEMPLATE_OUTPUTS must be a JSON object")
	}
	metadata, err := parseFoundationOutputMetadata()
	if err != nil {
		return tftypes.Value{}, nil, err
	}
	var tfType tftypes.Type
	if allowLegacyInference && helpers.OptionalEnv("CODER_FOUNDATION_OUTPUT_METADATA") == "" {
		// The legacy data source accepted values-only snapshots from servers
		// which excluded sensitive outputs. Only this compatibility path infers
		// types; coder_foundation requires exact types for nonempty snapshots.
		impliedType, err := ctyjson.ImpliedType(outputsJSON)
		if err != nil {
			return tftypes.Value{}, nil, errors.New("Unable to determine shared resource output types")
		}
		typeJSON, err := ctyjson.MarshalType(impliedType)
		if err != nil {
			return tftypes.Value{}, nil, errors.New("Unable to determine shared resource output types")
		}
		tfType, err = tftypes.ParseJSONType(typeJSON)
		if err != nil {
			return tftypes.Value{}, nil, errors.New("Unable to determine shared resource output types")
		}
	} else {
		if len(metadata) != len(outputs) {
			return tftypes.Value{}, nil, errors.New("Foundation output metadata must describe exactly the outputs in CODER_TEMPLATE_OUTPUTS")
		}
		attributeTypes := make(map[string]tftypes.Type, len(outputs))
		for name := range outputs {
			output, ok := metadata[name]
			if !ok {
				return tftypes.Value{}, nil, errors.New("Foundation output metadata must describe exactly the outputs in CODER_TEMPLATE_OUTPUTS")
			}
			outputType, err := tftypes.ParseJSONType(output.Type)
			if err != nil {
				return tftypes.Value{}, nil, errors.New("Unable to determine Foundation output types")
			}
			attributeTypes[name] = outputType
		}
		tfType = tftypes.Object{AttributeTypes: attributeTypes}
	}
	tfValue, err := tftypes.ValueFromJSON(outputsJSON, tfType)
	if err != nil {
		// Decoder errors may contain secret values. Do not include them.
		return tftypes.Value{}, nil, errors.New("Foundation output values do not match their Terraform types")
	}
	return tfValue, metadata, nil
}

func foundationOutputValues(ctx context.Context, allowLegacyInference bool) (types.Dynamic, error) {
	tfValue, _, err := foundationOutputSnapshot(allowLegacyInference)
	if err != nil {
		return types.DynamicNull(), err
	}
	return dynamicOutputValue(ctx, tfValue)
}

func splitFoundationOutputValues(ctx context.Context) (types.Dynamic, types.Dynamic, error) {
	snapshot, metadata, err := foundationOutputSnapshot(false)
	if err != nil {
		return types.DynamicNull(), types.DynamicNull(), err
	}
	var outputs map[string]tftypes.Value
	if err := snapshot.As(&outputs); err != nil {
		return types.DynamicNull(), types.DynamicNull(), errors.New("Unable to convert Foundation output values")
	}
	publicTypes, sensitiveTypes := make(map[string]tftypes.Type), make(map[string]tftypes.Type)
	publicValues, sensitiveValues := make(map[string]tftypes.Value), make(map[string]tftypes.Value)
	for name, value := range outputs {
		if *metadata[name].Sensitive {
			sensitiveTypes[name], sensitiveValues[name] = value.Type(), value
		} else {
			publicTypes[name], publicValues[name] = value.Type(), value
		}
	}
	public, err := dynamicOutputValue(ctx, tftypes.NewValue(tftypes.Object{AttributeTypes: publicTypes}, publicValues))
	if err != nil {
		return types.DynamicNull(), types.DynamicNull(), err
	}
	sensitive, err := dynamicOutputValue(ctx, tftypes.NewValue(tftypes.Object{AttributeTypes: sensitiveTypes}, sensitiveValues))
	if err != nil {
		return types.DynamicNull(), types.DynamicNull(), err
	}
	return public, sensitive, nil
}

func dynamicOutputValue(ctx context.Context, tfValue tftypes.Value) (types.Dynamic, error) {
	value, err := types.DynamicType.ValueFromTerraform(ctx, tfValue)
	if err != nil {
		return types.DynamicNull(), errors.New("Unable to convert Foundation output values")
	}
	return value.(types.Dynamic), nil
}
