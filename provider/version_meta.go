package provider

// VersionMeta provides utilities for managing version metadata in resources.
// Resources and attributes can specify their minimum Coder version requirements
// using these utilities.

// MinCoderVersion is a helper function that returns a formatted version string
// for use in resource descriptions. This ensures consistent formatting.
func MinCoderVersion(version string) string {
	return "@minCoderVersion:" + version
}

// Common version constants for frequently used versions
const (
	// V2_18_0 is the base requirement for terraform-provider-coder v2.0+
	V2_18_0 = "v2.18.0"
	
	// V2_16_0 introduced the hidden attribute for apps
	V2_16_0 = "v2.16.0"
	
	// V2_21_0 introduced devcontainer support
	V2_21_0 = "v2.21.0"
	
	// V2_24_0 introduced AI task resources
	V2_24_0 = "v2.24.0"
)
