package interfaces

import (
	"context"
)

type StorageService interface {
	DeleteFile(ctx context.Context, publicID string) error
	GenerateUploadSignature(params map[string]interface{}) (map[string]interface{}, error)
	UploadFile(ctx context.Context, file interface{}, folder string) (url string, publicID string, err error)
	RenameFile(ctx context.Context, oldPublicID string, newPublicID string) (url string, publicID string, err error)
	DeleteFilesByPrefix(ctx context.Context, prefix string) error
	ListFolders(ctx context.Context, prefix string) ([]string, error)
}
