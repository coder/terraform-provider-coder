# Resource Version Requirements

This document explains how to maintain version requirements for terraform-provider-coder resources.

## Overview

The `resource_versions.go` file contains a registry of minimum Coder version requirements for each resource and data source. This information is automatically injected into the documentation during the build process.

## Adding a New Resource

When adding a new resource or data source, you MUST:

1. Add an entry to the `ResourceVersions` map in `resource_versions.go`
2. Set the minimum Coder version that supports this resource
3. Run `make gen` to regenerate the documentation

### Example

```go
var ResourceVersions = map[string]string{
    // ... existing entries ...
    "coder_new_resource": "v2.25.0", // Add your new resource here
}
```

## Version Guidelines

- **Default minimum**: v2.18.0 (the base requirement for terraform-provider-coder v2.0+)
- **New features**: Use the Coder version where the feature was first introduced
- **Attribute-level requirements**: Use the `AttributeVersions` map for attributes added after the resource itself

## How It Works

1. `terraform-plugin-docs` generates the initial documentation
2. Our custom `docsgen` script reads the version registry
3. Version notes are automatically added to each resource's documentation
4. The version compatibility matrix in `index.md` is maintained manually in the template

## Testing

After adding or updating version information:

1. Run `make gen`
2. Check that the generated docs include the version note
3. Verify the note appears after the resource description

## Common Issues

- **Missing version error**: If you see "no version information found", add the resource to `ResourceVersions`
- **Duplicate notes**: If a resource already has version info in its Description, remove it and rely on the registry
