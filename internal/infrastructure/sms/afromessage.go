package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type SendRequest struct {
	From     string `json:"from,omitempty"`
	Sender   string `json:"sender,omitempty"`
	To       string `json:"to"`
	Message  string `json:"message"`
	Callback string `json:"callback,omitempty"`
}

type SendResponse struct {
	MessageID string `json:"message_id"`
	Status    string `json:"status"`
}

type AfroMessageResponse struct {
	Acknowledge string            `json:"acknowledge"`
	Response    *StatusData       `json:"response,omitempty"`
	MessageID   string            `json:"message_id,omitempty"` // For send response
	Errors      map[string]string `json:"errors,omitempty"`
}

type StatusData struct {
	MessageID   string `json:"messageId"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

type SMSClient interface {
	Send(ctx context.Context, req SendRequest) (*SendResponse, error)
	GetStatus(ctx context.Context, messageID string) (*StatusData, error)
}

type afroMessageClient struct {
	apiKey       string
	senderName   string
	identifierID string
	baseURL      string
	httpClient   *http.Client
}

func NewAfroMessageClient() SMSClient {
	baseURL := os.Getenv("AFROMESSAGE_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.afromessage.com"
	}

	apiKey := os.Getenv("AFROMESSAGE_API_KEY")

	return &afroMessageClient{
		apiKey:       apiKey,
		senderName:   os.Getenv("AFROMESSAGE_SENDER_NAME"),
		identifierID: os.Getenv("AFROMESSAGE_IDENTIFIER_ID"),
		baseURL:      baseURL,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *afroMessageClient) Send(ctx context.Context, req SendRequest) (*SendResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("SMS provider not configured: missing API key")
	}

	if req.Sender == "" {
		req.Sender = c.senderName
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/send", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var afroResp AfroMessageResponse
	if err := json.Unmarshal(respBody, &afroResp); err != nil {
		return nil, fmt.Errorf("failed to parse AfroMessage response: %w", err)
	}

	if afroResp.Acknowledge != "success" {
		return nil, fmt.Errorf("AfroMessage error: %v", afroResp.Errors)
	}

	return &SendResponse{
		MessageID: afroResp.MessageID,
		Status:    "Sent",
	}, nil
}

func (c *afroMessageClient) GetStatus(ctx context.Context, messageID string) (*StatusData, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("SMS provider not configured: missing API key")
	}

	// AfroMessage rate limit: 1 request every 2 seconds
	time.Sleep(2 * time.Second)

	url := fmt.Sprintf("%s/api/status?id=%s", c.baseURL, messageID)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var afroResp AfroMessageResponse
	if err := json.Unmarshal(respBody, &afroResp); err != nil {
		return nil, fmt.Errorf("failed to parse AfroMessage status response: %w", err)
	}

	if afroResp.Acknowledge != "success" {
		return nil, fmt.Errorf("AfroMessage status error: %v", afroResp.Errors)
	}

	return afroResp.Response, nil
}

// Mock implementation for testing
type mockSMSClient struct {
	SentMessages []SendRequest
}

func NewMockSMSClient() SMSClient {
	return &mockSMSClient{
		SentMessages: []SendRequest{},
	}
}

func (m *mockSMSClient) Send(ctx context.Context, req SendRequest) (*SendResponse, error) {
	m.SentMessages = append(m.SentMessages, req)
	return &SendResponse{
		MessageID: "mock-id-" + req.To,
		Status:    "Sent (Mock)",
	}, nil
}

func (m *mockSMSClient) GetStatus(ctx context.Context, messageID string) (*StatusData, error) {
	return &StatusData{
		MessageID:   messageID,
		Status:      "DELIVRD",
		Description: "Delivered (Mock)",
	}, nil
}
