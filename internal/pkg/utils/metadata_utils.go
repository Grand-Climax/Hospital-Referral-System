package utils

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"strings"

	"github.com/dslipak/pdf"
	"github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/tag"
	_ "golang.org/x/image/webp"
)

// ExtractMetadata extracts various metadata from an io.Reader based on the file type.
func ExtractMetadata(r io.Reader, fileName, contentType string, fileSize int64) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})
	lowerName := strings.ToLower(fileName)

	isDicom := strings.Contains(contentType, "dicom") || strings.HasSuffix(lowerName, ".dcm")
	isPdf := strings.Contains(contentType, "pdf") || strings.HasSuffix(lowerName, ".pdf")

	if isDicom {
		// DICOM Extraction (Medical Imaging)
		dataset, err := dicom.Parse(r, 0, nil)
		if err == nil {
			if elem, err := dataset.FindElementByTag(tag.Modality); err == nil {
				metadata["modality"] = elem.Value.String()
			}
			if elem, err := dataset.FindElementByTag(tag.InstitutionName); err == nil {
				metadata["institution_name"] = elem.Value.String()
			}
			if elem, err := dataset.FindElementByTag(tag.SeriesDescription); err == nil {
				metadata["series_description"] = elem.Value.String()
			}
			if elem, err := dataset.FindElementByTag(tag.StudyDate); err == nil {
				metadata["study_date"] = elem.Value.String()
			}
			if elem, err := dataset.FindElementByTag(tag.PatientName); err == nil {
				metadata["patient_name"] = elem.Value.String()
			}
			metadata["is_medical"] = true
		}
	} else if isPdf {
		// PDF Extraction (Lab Reports / Summaries)
		if ra, ok := r.(io.ReaderAt); ok && fileSize > 0 {
			p, err := pdf.NewReader(ra, fileSize)
			if err == nil {
				info := p.Trailer().Key("Info")
				if info.Kind() == pdf.Dict {
					if author := info.Key("Author"); author.Kind() == pdf.String {
						metadata["author"] = author.String()
					}
					if creator := info.Key("Creator"); creator.Kind() == pdf.String {
						metadata["creator"] = creator.String()
					}
					if creationDate := info.Key("CreationDate"); creationDate.Kind() == pdf.String {
						metadata["creation_date"] = creationDate.String()
					}
				}
				metadata["page_count"] = p.NumPage()
			}
		}
	} else {
		// Standard Image Dimensions
		config, format, err := image.DecodeConfig(r)
		if err == nil {
			metadata["width"] = config.Width
			metadata["height"] = config.Height
			metadata["format"] = format
		}
	}

	return metadata, nil
}
