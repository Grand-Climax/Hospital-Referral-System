package storage

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api"
	"github.com/cloudinary/cloudinary-go/v2/api/admin"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	iinfra "Hospital-Referral-System/internal/domain/interfaces/infrastructure"
)

type cloudinaryStorage struct {
	client    *cloudinary.Cloudinary
	apiKey    string
	apiSecret string
}

func NewCloudinaryStorage(cloudName, apiKey, apiSecret string) (iinfra.StorageService, error) {
	if cloudName == "" || apiKey == "" || apiSecret == "" {
		return nil, fmt.Errorf("cloudinary credentials are required")
	}
	cld, err := cloudinary.NewFromParams(cloudName, apiKey, apiSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Cloudinary: %w", err)
	}
	return &cloudinaryStorage{client: cld, apiKey: apiKey, apiSecret: apiSecret}, nil
}

func (s *cloudinaryStorage) GenerateUploadSignature(params map[string]interface{}) (map[string]interface{}, error) {
	timestamp := time.Now().Unix()
	params["timestamp"] = timestamp
	
	values := url.Values{}
	for k, v := range params {
		values.Set(k, fmt.Sprintf("%v", v))
	}

	signature, err := api.SignParameters(values, s.apiSecret)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"signature":  signature,
		"timestamp":  timestamp,
		"api_key":    s.apiKey,
		"cloud_name": s.client.Config.Cloud.CloudName,
		"folder":     params["folder"],
	}, nil
}

func (s *cloudinaryStorage) DeleteFile(ctx context.Context, publicID string) error {
	if publicID == "" {
		return nil
	}
	_, err := s.client.Upload.Destroy(ctx, uploader.DestroyParams{
		PublicID: publicID,
	})
	return err
}
func (s *cloudinaryStorage) RenameFile(ctx context.Context, oldPublicID string, newPublicID string) (string, string, error) {
	renameResult, err := s.client.Upload.Rename(ctx, uploader.RenameParams{
		FromPublicID: oldPublicID,
		ToPublicID:   newPublicID,
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to rename file: %w", err)
	}

	return renameResult.SecureURL, renameResult.PublicID, nil
}

func (s *cloudinaryStorage) DeleteFilesByPrefix(ctx context.Context, prefix string) error {
	if prefix == "" {
		return fmt.Errorf("prefix is required")
	}
	_, err := s.client.Admin.DeleteAssetsByPrefix(ctx, admin.DeleteAssetsByPrefixParams{
		Prefix: []string{prefix},
	})
	if err != nil {
		return fmt.Errorf("failed to delete assets by prefix %s: %w", prefix, err)
	}
	
	// Try to delete the folder itself (fails if not empty, but good to clean up)
	_, _ = s.client.Admin.DeleteFolder(ctx, admin.DeleteFolderParams{Folder: prefix})
	return nil
}

func (s *cloudinaryStorage) ListFolders(ctx context.Context, prefix string) ([]string, error) {
	resp, err := s.client.Admin.SubFolders(ctx, admin.SubFoldersParams{
		Folder: prefix,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list subfolders for %s: %w", prefix, err)
	}

	var folders []string
	for _, f := range resp.Folders {
		folders = append(folders, f.Path)
	}
	return folders, nil
}

func (s *cloudinaryStorage) UploadFile(ctx context.Context, file interface{}, folder string) (string, string, error) {
	uploadResult, err := s.client.Upload.Upload(ctx, file, uploader.UploadParams{
		Folder: folder,
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to upload file: %w", err)
	}

	return uploadResult.SecureURL, uploadResult.PublicID, nil
}
