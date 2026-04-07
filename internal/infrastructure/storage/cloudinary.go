package storage

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
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
func (s *cloudinaryStorage) VerifyWebhookSignature(headers map[string]string, body []byte) (bool, error) {
	signature := headers["X-Cld-Signature"]
	timestamp := headers["X-Cld-Timestamp"]

	if signature == "" || timestamp == "" {
		return false, fmt.Errorf("missing signature or timestamp")
	}

	// Cloudinary signature verification: SHA1(body + timestamp + secret)
	toSign := string(body) + timestamp + s.apiSecret
	hash := sha1.New()
	hash.Write([]byte(toSign))
	expectedSignature := hex.EncodeToString(hash.Sum(nil))

	return expectedSignature == signature, nil
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
