# Version Metadata Documentation

This document explains how to add version requirements to resources and attributes in the terraform-provider-coder.

## Overview

Version information is embedded directly in resource and attribute descriptions using special markers. The documentation generation process automatically extracts this information and formats it appropriately.

## Adding Version Requirements

### For Resources

Add a version marker to the resource's `Description` field:

```go
Description: "Your resource description. @minCoderVersion:v2.21.0",
```

The marker will be automatically removed from the generated docs and replaced with a formatted note.

### For Attributes

Add a version marker to the attribute's `Description` field:

```go
"my_attribute": {
    Type:        schema.TypeString,
    Description: "Attribute description. @since:v2.16.0",
    Optional:    true,
},
```

This will result in the documentation showing: `- my_attribute (String) Attribute description. *(since v2.16.0)*`

## Version Marker Formats

You can use either format:
- `@minCoderVersion:vX.Y.Z` - For resources
- `@since:vX.Y.Z` - For attributes

Both formats are recognized and processed the same way.

## How It Works

1. **During Development**: Add version markers to descriptions
2. **During Doc Generation**: 
   - `terraform-plugin-docs` generates initial documentation
   - Our custom `docsgen` script:
     - Extracts version information from descriptions
     - Adds formatted version notes to resources
     - Adds inline version markers to attributes
     - Cleans up the version patterns from descriptions

## Examples

### Resource Example

```go
func myNewResource() *schema.Resource {
    return &schema.Resource{
        Description: "Manages a new Coder feature. @minCoderVersion:v2.25.0",
        // ... rest of resource definition
    }
}
```

Results in documentation with:
```markdown
# coder_my_new (Resource)

Manages a new Coder feature.

~> **Note:** This resource requires [Coder v2.25.0](https://github.com/coder/coder/releases/tag/v2.25.0) or later.
```

### Attribute Example

```go
"advanced_option": {
    Type:        schema.TypeBool,
    Description: "Enable advanced features. @since:v2.22.0",
    Optional:    true,
},
```

Results in documentation with:
```markdown
- `advanced_option` (Boolean) Enable advanced features. *(since v2.22.0)*
```

## Default Versions

- Resources without version markers default to `v2.18.0` (the base requirement)
- Attributes without version markers don't show version information
- Special resources have hardcoded defaults:
  - `coder_devcontainer`: v2.21.0
  - `coder_ai_task`: v2.24.0

## Best Practices

1. **Always add version markers** when creating new resources or attributes
2. **Use semantic versioning** (vX.Y.Z format)
3. **Test documentation generation** with `make gen` after adding markers
4. **Keep descriptions concise** - the version marker is removed from the final docs
