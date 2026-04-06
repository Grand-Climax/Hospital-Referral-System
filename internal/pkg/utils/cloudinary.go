package utils

import (
	"strings"
)

// OptimizeCloudinaryURL injects q_auto and f_auto transformations into a Cloudinary URL.
// Example: https://res.cloudinary.com/demo/image/upload/sample.jpg
// Result: https://res.cloudinary.com/demo/image/upload/q_auto,f_auto/sample.jpg
func OptimizeCloudinaryURL(url string) string {
	if url == "" || !strings.Contains(url, "cloudinary.com") {
		return url
	}

	// We look for "/upload/" and inject transformations after it
	// If transformations already exist (e.g. /upload/w_100/...), we append to them
	parts := strings.Split(url, "/upload/")
	if len(parts) != 2 {
		return url
	}

	// Check if there are already transformations
	subParts := strings.Split(parts[1], "/")
	if len(subParts) > 1 && (strings.Contains(subParts[0], "_") || strings.Contains(subParts[0], ",")) {
		// Existing transformations found, prepend our defaults if not present
		existing := subParts[0]
		if !strings.Contains(existing, "q_auto") {
			existing = "q_auto," + existing
		}
		if !strings.Contains(existing, "f_auto") {
			existing = "f_auto," + existing
		}
		return parts[0] + "/upload/" + existing + "/" + strings.Join(subParts[1:], "/")
	}

	// No existing transformations
	return parts[0] + "/upload/q_auto,f_auto/" + parts[1]
}
