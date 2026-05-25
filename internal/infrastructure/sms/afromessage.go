package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
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
	Acknowledge string `json:"acknowledge,omitempty"`
}

type BulkSendRequest struct {
	To       []BulkRecipient `json:"to"`
	From     string          `json:"from,omitempty"`
	Sender   string          `json:"sender,omitempty"`
	Campaign string          `json:"campaign,omitempty"`
}

type BulkRecipient struct {
	To      string `json:"to"`
	Message string `json:"message"`
}

type BulkSendResponse struct {
	Acknowledge string `json:"acknowledge"`
	Response    struct {
		Status   string `json:"status"`
		BatchID  string `json:"batch_id"`
		Messages []struct {
			To        string `json:"to"`
			MessageID string `json:"message_id"`
		} `json:"messages"`
		Errors []string `json:"errors,omitempty"`
	} `json:"response"`
}

type SendResponseData struct {
	MessageID string `json:"message_id"`
	Status    string `json:"status"`
}

type AfroMessageResponse struct {
	Acknowledge string            `json:"acknowledge"`
	Response    json.RawMessage   `json:"response,omitempty"`
	MessageID   string            `json:"message_id,omitempty"` // Fallback for some API versions
	Errors      map[string]string `json:"errors,omitempty"`
}

type StatusData struct {
	MessageID   string `json:"messageId"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

type SMSClient interface {
	Send(ctx context.Context, req SendRequest) (*SendResponse, error)
	SendBulk(ctx context.Context, req BulkSendRequest) (*BulkSendResponse, error)
	GetStatus(ctx context.Context, messageID string) (*StatusData, error)
}

type afroMessageClient struct {
	apiKey        string
	defaultSender string
	defaultFrom   string
	baseURL       string
	httpClient    *http.Client
}

func NewAfroMessageClient() SMSClient {
	// TrimSpace defends against CRLF-terminated secrets that were
	// written on Windows or via tooling that appended a trailing \r\n.
	// Without this, os.Getenv returns "https://api.afromessage.com\r\n"
	// and net/http.NewRequest rejects the URL with "invalid control
	// character in URL".
	apiKey := strings.TrimSpace(os.Getenv("AFROMESSAGE_API_KEY"))
	if apiKey == "" {
		log.Println("Warning: AFROMESSAGE_API_KEY is not configured. Falling back to log-based mockSMSClient.")
		return NewMockSMSClient()
	}

	baseURL := "https://api.afromessage.com"

	return &afroMessageClient{
		apiKey:        apiKey,
		defaultSender: strings.TrimSpace(os.Getenv("AFROMESSAGE_SENDER_NAME")),
		defaultFrom:   strings.TrimSpace(os.Getenv("AFROMESSAGE_IDENTIFIER_ID")),
		baseURL:       baseURL,
		httpClient:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *afroMessageClient) Send(ctx context.Context, req SendRequest) (*SendResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("SMS provider not configured: missing API key")
	}

	// Prepare payload
	payload := map[string]interface{}{
		"to":      req.To,
		"message": req.Message,
	}

	// Use from/identifierID if provided
	fromID := req.From
	if fromID == "" {
		fromID = c.defaultFrom
	}
	if fromID != "" {
		payload["from"] = fromID
	}

	// Use sender name if provided
	sender := req.Sender
	if sender == "" {
		sender = c.defaultSender
	}
	if sender != "" {
		payload["sender"] = sender
	}

	if req.Callback != "" {
		payload["callback"] = req.Callback
	}

	body, err := json.Marshal(payload)
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

	// Even if AfroMessage returns 4xx/5xx, the body is usually JSON
	// describing the rejection. We parse-best-effort and surface the
	// raw body in the error so operators can see exactly what the
	// provider said. The previous behaviour ("AfroMessage error: map[]")
	// dropped every actionable detail when Errors was empty.
	var afroResp AfroMessageResponse
	parseErr := json.Unmarshal(respBody, &afroResp)

	bodySnippet := string(respBody)
	if len(bodySnippet) > 512 {
		bodySnippet = bodySnippet[:512] + "...[truncated]"
	}

	if parseErr != nil {
		log.Printf("afromessage send: parse error http=%d body=%q to=%s",
			resp.StatusCode, bodySnippet, req.To)
		return nil, fmt.Errorf("AfroMessage parse error: http=%d body=%s", resp.StatusCode, bodySnippet)
	}

	if afroResp.Acknowledge != "success" {
		log.Printf("afromessage send REJECTED http=%d acknowledge=%q errors=%v body=%q to=%s",
			resp.StatusCode, afroResp.Acknowledge, afroResp.Errors, bodySnippet, req.To)
		return nil, fmt.Errorf(
			"AfroMessage rejected send: http=%d acknowledge=%q errors=%v body=%s",
			resp.StatusCode, afroResp.Acknowledge, afroResp.Errors, bodySnippet,
		)
	}

	// Extract message ID from polymorphic response
	messageID := afroResp.MessageID
	if messageID == "" && afroResp.Response != nil {
		var sendData SendResponseData
		if err := json.Unmarshal(afroResp.Response, &sendData); err == nil {
			messageID = sendData.MessageID
		}
	}

	return &SendResponse{
		MessageID: messageID,
		Status:    "Sent",
	}, nil
}

func (c *afroMessageClient) SendBulk(ctx context.Context, req BulkSendRequest) (*BulkSendResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("SMS provider not configured: missing API key")
	}

	// Set default values if not provided
	if req.From == "" {
		req.From = c.defaultFrom
	}
	if req.Sender == "" {
		req.Sender = c.defaultSender
	}
	if req.Campaign == "" {
		req.Campaign = fmt.Sprintf("ReferralHub-%s", time.Now().Format("20060102-150405"))
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/bulk_send", c.baseURL)
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

	var bulkResp BulkSendResponse
	if err := json.NewDecoder(resp.Body).Decode(&bulkResp); err != nil {
		return nil, err
	}

	return &bulkResp, nil
}

func (c *afroMessageClient) GetStatus(ctx context.Context, messageID string) (*StatusData, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("SMS provider not configured: missing API key")
	}

	// AfroMessage rate limit: 1 request every 2 seconds
	time.Sleep(2 * time.Second)

	urlPath := fmt.Sprintf("%s/api/status?id=%s", c.baseURL, url.QueryEscape(messageID))
	httpReq, err := http.NewRequestWithContext(ctx, "GET", urlPath, nil)
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

	if afroResp.Response == nil {
		return nil, fmt.Errorf("AfroMessage status response is empty")
	}

	var statusData StatusData
	if err := json.Unmarshal(afroResp.Response, &statusData); err != nil {
		return nil, fmt.Errorf("failed to parse StatusData: %w", err)
	}

	return &statusData, nil
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
	log.Printf("[DEV SMS] Sending SMS to %s | Message: %s", req.To, req.Message)
	return &SendResponse{
		MessageID: "mock-id-" + req.To,
		Status:    "Sent (Mock)",
	}, nil
}

func (m *mockSMSClient) SendBulk(ctx context.Context, req BulkSendRequest) (*BulkSendResponse, error) {
	resp := &BulkSendResponse{
		Acknowledge: "success",
	}
	resp.Response.Status = "Send in progress (Mock)"
	resp.Response.BatchID = "mock-batch-" + uuid.NewString()

	for _, rec := range req.To {
		m.SentMessages = append(m.SentMessages, SendRequest{
			To:      rec.To,
			Message: rec.Message,
		})
		log.Printf("[DEV SMS BULK] Sending SMS to %s | Message: %s", rec.To, rec.Message)
		resp.Response.Messages = append(resp.Response.Messages, struct {
			To        string `json:"to"`
			MessageID string `json:"message_id"`
		}{
			To:        rec.To,
			MessageID: "mock-id-" + rec.To,
		})
	}
	return resp, nil
}

func (m *mockSMSClient) GetStatus(ctx context.Context, messageID string) (*StatusData, error) {
	return &StatusData{
		MessageID:   messageID,
		Status:      "DELIVRD",
		Description: "Delivered (Mock)",
	}, nil
}
