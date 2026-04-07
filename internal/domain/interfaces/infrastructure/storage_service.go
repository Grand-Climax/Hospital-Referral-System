package infrastructure

import (
	"context"
)

type StorageService interface {
	DeleteFile(ctx context.Context, publicID string) error
	GenerateUploadSignature(params map[string]interface{}) (map[string]interface{}, error)
	VerifyWebhookSignature(headers map[string]string, body []byte) (bool, error)
	UploadFile(ctx context.Context, file interface{}, folder string) (url string, publicID string, err error)
}
