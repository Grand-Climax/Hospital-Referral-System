package infrastructure

import (
	"context"
)

type StorageService interface {
	DeleteFile(ctx context.Context, publicID string) error
	GenerateUploadSignature(params map[string]interface{}) (map[string]interface{}, error)
}
