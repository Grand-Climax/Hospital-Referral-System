package storage

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"Hospital-Referral-System/internal/domain/interfaces/infrastructure"
)

type cloudinaryStorage struct {
	client    *cloudinary.Cloudinary
	apiKey    string
	apiSecret string
}

func NewCloudinaryStorage(cloudName, apiKey, apiSecret string) (infrastructure.StorageService, error) {
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
