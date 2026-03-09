package service

import (
	"bytes"
	"encoding/json"
	"flows/internal/domain"
	"fmt"
	"net/http"
	"os"
	"time"
)

type StatusUpdate struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	UUID      string `json:"uuid"`
}

type SubscriptionClient struct {
	EndpointURL string
	HTTPClient  *http.Client
}

func NewSubscriptionClientFromEnv() *SubscriptionClient {
	url := os.Getenv("SUBSCRIPTION_UPDATE_URL")
	if url == "" {
		return nil
	}
	return &SubscriptionClient{
		EndpointURL: url,
		HTTPClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *SubscriptionClient) SendStatus(exec *domain.Execution) {
	if c == nil || c.EndpointURL == "" || exec == nil {
		return
	}

	status := mapExecutionStatus(exec.Status)

	payload := StatusUpdate{
		Status:    status,
		Timestamp: time.Now().Format(time.RFC3339),
		UUID:      exec.ID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	req, err := http.NewRequest("POST", c.EndpointURL, bytes.NewBuffer(body))
	if err != nil {
		fmt.Printf("error creating subscription request: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		fmt.Printf("subscription request failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		fmt.Printf("subscription service returned status %d\n", resp.StatusCode)
	}
}

func mapExecutionStatus(s domain.ExecutionStatus) string {
	switch s {
	case domain.StatusPending:
		return "PENDING"
	case domain.StatusCompleted:
		return "COMPLETED"
	case domain.StatusFailed:
		return "FAILED"
	case domain.StatusRunning, domain.StatusWaiting, domain.StatusSuspended:
		return "RUNNING"
	default:
		return "RUNNING"
	}
}
