package proc

import (
	"regexp"
	"strconv"
	"strings"
)

// versionRegex matches Go version strings like "go1.18" or "go1.18.3"
var versionRegex = regexp.MustCompile(`go(\d+\.\d+)`)

// waitReasonRegistry holds version-specific wait reason mappings
type waitReasonRegistry struct {
	versions map[string]map[uint8]string
}

var registry = &waitReasonRegistry{
	versions: make(map[string]map[uint8]string),
}

func init() {
	registry.Register("1.24", waitReasonMap1_24)
	// 23 no change
	registry.Register("1.23", waitReasonMap1_22)
	registry.Register("1.22", waitReasonMap1_22)
	registry.Register("1.21", waitReasonMap1_21)
	registry.Register("1.20", waitReasonMap1_20)
	registry.Register("1.18", waitReasonMap1_18)
}

// GetWaitReasonMap returns the wait reason map for the given version
func (r *waitReasonRegistry) GetWaitReasonMap(version string) map[uint8]string {
	normalized := normalizeVersion(version)

	// 1. Exact match
	if m, ok := r.versions[normalized]; ok {
		return m
	}

	// 2. Major version match (e.g. 1.18.3 matches 1.18)
	parts := strings.Split(normalized, ".")
	if len(parts) >= 2 {
		major := parts[0] + "." + parts[1]
		if m, ok := r.versions[major]; ok {
			return m
		}
	}

	// 3. Return latest registered version (with proper version comparison)
	if len(r.versions) > 0 {
		// Find highest version number
		var maxVersion string
		for v := range r.versions {
			if compareVersions(v, maxVersion) > 0 {
				maxVersion = v
			}
		}
		return r.versions[maxVersion]
	}

	// 4. Fallback to empty map
	return make(map[uint8]string)
}

// Register adds a new version's wait reason mapping
func (r *waitReasonRegistry) Register(version string, reasons map[uint8]string) {
	normalized := normalizeVersion(version)
	r.versions[normalized] = reasons
}

// compareVersions compares two version strings, returns:
// -1 if v1 < v2
// 0 if v1 == v2
// 1 if v1 > v2
func compareVersions(v1, v2 string) int {
	if v1 == v2 {
		return 0
	}

	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	for i := 0; i < len(parts1) && i < len(parts2); i++ {
		num1, _ := strconv.Atoi(parts1[i])
		num2, _ := strconv.Atoi(parts2[i])

		if num1 < num2 {
			return -1
		}
		if num1 > num2 {
			return 1
		}
	}

	// If common parts are equal, longer version is greater
	if len(parts1) < len(parts2) {
		return -1
	}
	if len(parts1) > len(parts2) {
		return 1
	}

	return 0
}

// normalizeVersion extracts the major.minor version from full version strings
func normalizeVersion(v string) string {
	v = strings.TrimPrefix(v, "go")
	if matches := versionRegex.FindStringSubmatch(v); len(matches) > 1 {
		return matches[1] // Returns "1.18" from "go1.18.3"
	}
	return v
}
