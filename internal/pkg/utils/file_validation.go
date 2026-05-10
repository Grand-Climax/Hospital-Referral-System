package utils

import (
	"fmt"
	"strings"
)

var AllowedExtensions = map[string]bool{
	".pdf":   true,
	".jpg":   true,
	".jpeg":  true,
	".png":   true,
	".bmp":   true,
	".tiff":  true,
	".tif":   true,
	".dcm":   true,
	".dicom": true,
	".nii":   true,
	".mha":   true,
	".mhd":   true,
}

// ValidateFileExtension checks if the file has an allowed extension.
// Returns the lowercased extension and nil if valid, or an error if not.
func ValidateFileExtension(fileName string) (string, error) {
	dotIdx := strings.LastIndex(fileName, ".")
	if dotIdx == -1 {
		return "", fmt.Errorf("file has no extension. Accepted types: PDF, JPEG, PNG, BMP, TIFF, DICOM, NIfTI")
	}
	
	ext := strings.ToLower(fileName[dotIdx:])
	if !AllowedExtensions[ext] {
		return ext, fmt.Errorf("file type %s is not allowed. Accepted types: PDF, JPEG, PNG, BMP, TIFF, DICOM, NIfTI", ext)
	}
	return ext, nil
}
