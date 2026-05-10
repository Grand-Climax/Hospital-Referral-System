package utils

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"strings"

	"github.com/ledongthuc/pdf"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/suyashkumar/dicom"
	dicomtag "github.com/suyashkumar/dicom/pkg/tag"
	"github.com/rwcarlsen/goexif/exif"
)

func ExtractMetadata(reader io.Reader, fileName, fileType string, fileSize int64) (map[string]interface{}, error) {
	// Read all data into memory (max 256KB limit for this function)
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	isPDF := strings.Contains(strings.ToLower(fileType), "pdf") || strings.HasSuffix(strings.ToLower(fileName), ".pdf")
	isDICOM := strings.Contains(strings.ToLower(fileType), "dicom") || strings.HasSuffix(strings.ToLower(fileName), ".dcm")
	isImage := strings.Contains(strings.ToLower(fileType), "image/") || 
		strings.HasSuffix(strings.ToLower(fileName), ".jpg") ||
		strings.HasSuffix(strings.ToLower(fileName), ".jpeg")

	if isPDF {
		return extractPDFMetadata(data)
	}
	if isDICOM {
		return extractDICOMMetadata(data)
	}
	if isImage {
		return extractImageMetadata(data)
	}

	return map[string]interface{}{}, nil
}

func extractPDFMetadata(data []byte) (map[string]interface{}, error) {
	reader := bytes.NewReader(data)
	
	// Use ledongthuc/pdf for reliable page counting
	pdfReader, err := pdf.NewReader(reader, int64(len(data)))
	pageCount := 0
	if err == nil {
		pageCount = pdfReader.NumPage()
	}

	metadata := map[string]interface{}{
		"page_count": pageCount,
	}

	// Use pdfcpu for other metadata
	reader2 := bytes.NewReader(data)
	conf := model.NewDefaultConfiguration()
	ctx, err2 := api.ReadContext(reader2, conf)
	if err2 != nil {
		if err != nil {
			// Both failed
			return map[string]interface{}{
				"page_count": 0,
				"error":      fmt.Sprintf("PDF parse failed: %v", err2),
			}, nil
		}
		// pdfcpu failed but ledongthuc succeeded with page count
		return metadata, nil
	}

	if ctx.Info != nil {
		if d, err := ctx.DereferenceDict(*ctx.Info); err == nil {
			if s := d.StringEntry("Title"); s != nil {
				metadata["title"] = *s
			}
			if s := d.StringEntry("Author"); s != nil {
				metadata["author"] = *s
			}
			if s := d.StringEntry("Subject"); s != nil {
				metadata["subject"] = *s
			}
			if s := d.StringEntry("Keywords"); s != nil {
				metadata["keywords"] = *s
			}
			if s := d.StringEntry("Creator"); s != nil {
				metadata["creator"] = *s
			}
			if s := d.StringEntry("Producer"); s != nil {
				metadata["producer"] = *s
			}
			if s := d.StringEntry("CreationDate"); s != nil {
				metadata["creation_date"] = *s
			}
			if s := d.StringEntry("ModDate"); s != nil {
				metadata["mod_date"] = *s
			}
		}
	}
	
	return metadata, nil
}

func extractDICOMMetadata(data []byte) (map[string]interface{}, error) {
	reader := bytes.NewReader(data)
	
	dataset, err := dicom.Parse(reader, int64(len(data)), nil)
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("DICOM parse failed: %v", err),
		}, nil
	}

	metadata := map[string]interface{}{
		"modality":           getDICOMString(dataset, dicomtag.Modality),
		"study_date":         getDICOMString(dataset, dicomtag.StudyDate),
		"patient_name":       getDICOMString(dataset, dicomtag.PatientName),
		"patient_id":         getDICOMString(dataset, dicomtag.PatientID),
		"institution_name":   getDICOMString(dataset, dicomtag.InstitutionName),
		"manufacturer":       getDICOMString(dataset, dicomtag.Manufacturer),
		"study_description":  getDICOMString(dataset, dicomtag.StudyDescription),
		"series_description": getDICOMString(dataset, dicomtag.SeriesDescription),
	}
	
	return metadata, nil
}

func getDICOMString(ds dicom.Dataset, t dicomtag.Tag) string {
	elem, err := ds.FindElementByTag(t)
	if err != nil {
		return ""
	}
	// Extract string value from DICOM element safely
	if elem.Value == nil {
		return ""
	}
	s := elem.Value.String()
	// Strip brackets if it's a slice string like "[VALUE]"
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	return s
}

func extractImageMetadata(data []byte) (map[string]interface{}, error) {
	metadata := map[string]interface{}{}

	// Try EXIF extraction first (best-effort)
	reader := bytes.NewReader(data)
	x, err := exif.Decode(reader)
	if err == nil {
		// Camera info
		if tag, err := x.Get(exif.Make); err == nil {
			val, _ := tag.StringVal()
			if val != "" {
				metadata["camera_make"] = val
			}
		}
		if tag, err := x.Get(exif.Model); err == nil {
			val, _ := tag.StringVal()
			if val != "" {
				metadata["camera_model"] = val
			}
		}
		if tag, err := x.Get(exif.DateTimeOriginal); err == nil {
			val, _ := tag.StringVal()
			if val != "" {
				metadata["date_taken"] = val
			}
		}
		// GPS
		if lat, lon, err2 := x.LatLong(); err2 == nil {
			metadata["gps_latitude"] = lat
			metadata["gps_longitude"] = lon
		}
	}
	// If EXIF fails, we don't return an error - we fall through to basic decoding.

	// Always extract basic image dimensions using Go's standard library
	reader2 := bytes.NewReader(data)
	config, format, err := image.DecodeConfig(reader2)
	if err == nil {
		metadata["format"] = format
		metadata["width"] = config.Width
		metadata["height"] = config.Height
	}

	// If we got absolutely nothing, add a note
	if len(metadata) == 0 {
		metadata["note"] = "No EXIF data and unable to decode image dimensions"
	}

	return metadata, nil
}
