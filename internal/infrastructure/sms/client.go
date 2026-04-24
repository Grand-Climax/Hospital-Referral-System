package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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
	MessageID   string            `json:"message_id,omitempty"`
	Errors      map[string]string `json:"errors,omitempty"`
}

type SMSClient interface {
	Send(ctx context.Context, req SendRequest) (*SendResponse, error)
}

type afroMessageClient struct {
	apiKey     string
	senderName string
	baseURL    string
	httpClient *http.Client
}

func NewAfroMessageClient() SMSClient {
	return &afroMessageClient{
		apiKey:     os.Getenv("AFROMESSAGE_API_KEY"),
		senderName: os.Getenv("AFROMESSAGE_SENDER_NAME"),
		baseURL:    "https://api.afromessage.com/api/send",
		httpClient: &http.Client{},
	}
}

func (c *afroMessageClient) Send(ctx context.Context, req SendRequest) (*SendResponse, error) {
	if req.Sender == "" {
		req.Sender = c.senderName
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewBuffer(body))
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
