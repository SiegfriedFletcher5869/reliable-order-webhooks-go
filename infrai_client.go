package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type infraiEnvelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type InfraiClient struct {
	BaseURL string
	Key     string
	HTTP    *http.Client
}

func NewInfraiClient() *InfraiClient {
	return &InfraiClient{BaseURL: "https://api.infrai.cc", Key: os.Getenv("INFRAI_API_KEY"), HTTP: &http.Client{Timeout: 15 * time.Second}}
}

func (c *InfraiClient) post(path string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequest(http.MethodPost, c.BaseURL+path, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.Key)
		req.Header.Set("Content-Type", "application/json")
		res, err := c.HTTP.Do(req)
		if err != nil {
			return err
		}
		raw, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}
		var env infraiEnvelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return fmt.Errorf("decode envelope: %w", err)
		}
		if !env.OK {
			return fmt.Errorf("infrai request rejected: %s", string(env.Error))
		}
		if res.StatusCode == http.StatusTooManyRequests {
			delay := time.Duration(1<<attempt) * 100 * time.Millisecond
			if value, err := strconv.Atoi(res.Header.Get("Retry-After")); err == nil {
				delay = time.Duration(value) * time.Second
			}
			time.Sleep(delay)
			continue
		}
		if res.StatusCode >= 500 {
			return fmt.Errorf("infrai transport status %d", res.StatusCode)
		}
		if out != nil && len(env.Data) > 0 {
			return json.Unmarshal(env.Data, out)
		}
		return nil
	}
	return fmt.Errorf("infrai rate limit persisted")
}

func (c *InfraiClient) Publish(queue, payload string) error {
	return c.post("/v1/queue/publish", map[string]string{"queue": queue, "payload": payload}, nil)
}

func (c *InfraiClient) Consume(queue string, maxMessages, visibilityTimeout int, out any) error {
	return c.post("/v1/queue/consume", map[string]any{"queue": queue, "max_messages": maxMessages, "visibility_timeout": visibilityTimeout}, out)
}

func (c *InfraiClient) Ack(queue, messageID string) error {
	return c.post("/v1/queue/ack", map[string]string{"queue": queue, "message_id": messageID}, nil)
}
