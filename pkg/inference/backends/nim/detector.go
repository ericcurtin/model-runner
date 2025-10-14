package nim

import (
	"strings"
)

// IsNIMReference checks if a model reference is a NIM container.
// NIM containers are typically hosted on nvcr.io/nim/* registry.
func IsNIMReference(reference string) bool {
	// Check if the reference is from NVIDIA's NIM registry
	if strings.Contains(reference, "nvcr.io/nim/") {
		return true
	}
	
	// Also check for variations without the full registry path
	if strings.HasPrefix(reference, "nim/") {
		return true
	}
	
	return false
}
