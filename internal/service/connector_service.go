package service

import (
	"bytes"
	"encoding/json"
	"flows/internal/domain"
	"flows/internal/dto"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"gorm.io/gorm"
)

type ConnectorService struct {
	DB *gorm.DB
}

func NewConnectorService(db *gorm.DB) *ConnectorService {
	return &ConnectorService{DB: db}
}

// ExecuteConnector performs the actual call to the external service
func (s *ConnectorService) ExecuteConnector(connectorID uint, input json.RawMessage, env string, stepConfig map[string]interface{}) (json.RawMessage, error) {
	// 1. Load Connector Definition
	var connector domain.Connector
	if err := s.DB.First(&connector, connectorID).Error; err != nil {
		return nil, fmt.Errorf("connector not found: %w", err)
	}

	// 2. Load Environment Config (Secrets)
	var config domain.ConnectorConfig
	// Default to 'development' if env not provided
	if env == "" {
		env = "development"
	}
	// In a real scenario, handle error gracefully if config missing
	s.DB.Where("connector_id = ? AND environment = ?", connectorID, env).First(&config)

	// 3. Parse Policy
	var policy dto.ConnectorPolicy
	if len(connector.Policy) > 0 {
		json.Unmarshal(connector.Policy, &policy)
	}
	// Defaults
	if policy.TimeoutMs == 0 {
		policy.TimeoutMs = 5000
	}
	if policy.MaxRetries == 0 {
		policy.MaxRetries = 1
	}

	// Determine Target URL
	targetURL := connector.BaseURL
	if stepConfig != nil {
		if urlOverride, ok := stepConfig["url"].(string); ok && urlOverride != "" {
			// Check if it's absolute or relative
			if strings.HasPrefix(urlOverride, "http://") || strings.HasPrefix(urlOverride, "https://") {
				targetURL = urlOverride
			} else {
				// Append to BaseURL
				targetURL = strings.TrimRight(targetURL, "/") + "/" + strings.TrimLeft(urlOverride, "/")
			}
		} else if route, ok := stepConfig["route"].(string); ok && route != "" {
			targetURL = strings.TrimRight(targetURL, "/") + "/" + strings.TrimLeft(route, "/")
		}
	}

	client := &http.Client{
		Timeout: time.Duration(policy.TimeoutMs) * time.Millisecond,
	}

	req, err := http.NewRequest("POST", targetURL, bytes.NewBuffer(input))
	if err != nil {
		return nil, err
	}

	// Override URL if present in input config (but input is raw message here)
	// We need a way to pass per-step config overrides.
	// Currently ExecuteConnector takes (connectorID, input, env).
	// But FlowManager has access to step.Config!
	// We should update ExecuteConnector signature or handle it inside FlowManager before calling.
	// OR: Parse input to check for special overrides? No, input is payload.

	// Better approach: Pass step.Config to ExecuteConnector.

	req.Header.Set("Content-Type", "application/json")

	// Inject Auth Headers from Config
	if len(config.Config) > 0 {
		var secrets map[string]string
		json.Unmarshal(config.Config, &secrets)

		if connector.AuthType == domain.AuthTypeAPIKey {
			// Example: Assume key is 'api_key' in secrets
			if key, ok := secrets["api_key"]; ok {
				req.Header.Set("Authorization", "Bearer "+key)
			}
		}
	}

	// 5. Execute with Retries
	var resp *http.Response
	var reqErr error

	for i := 0; i <= policy.MaxRetries; i++ {
		// Log Request Dump
		reqDump, _ := httputil.DumpRequestOut(req, true)
		log.Printf("\n--- [ConnectorService] Sending Request (Attempt %d/%d) ---\n%s\n----------------------------------------------------", i+1, policy.MaxRetries+1, string(reqDump))

		resp, reqErr = client.Do(req)
		if reqErr == nil && resp.StatusCode < 500 {
			break // Success or non-retriable error
		}

		if i < policy.MaxRetries {
			log.Printf("[ConnectorService] Request failed, retrying in %dms...", policy.RetryBackoffMs)
			time.Sleep(time.Duration(policy.RetryBackoffMs) * time.Millisecond)
		}
	}

	if reqErr != nil {
		return nil, fmt.Errorf("request failed after retries: %w", reqErr)
	}
	defer resp.Body.Close()

	// Log Response Dump (needs to read body first)
	// We read body later anyway, so let's use DumpResponse but be careful with body reading
	respDump, _ := httputil.DumpResponse(resp, false) // false = don't read body yet to avoid consuming it
	log.Printf("\n--- [ConnectorService] Received Response ---\n%s\n(Body will be read separately)\n------------------------------------------", string(respDump))

	// 6. Read Response
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		// If it's not a JSON object (e.g. array or plain text), wrap it
		var arrayResult []interface{}
		if err2 := json.Unmarshal(bodyBytes, &arrayResult); err2 == nil {
			result = map[string]interface{}{"data": arrayResult}
		} else {
			// Plain text or invalid JSON
			result = map[string]interface{}{"raw_response": string(bodyBytes)}
		}
	}

	// Add metadata
	result["_status_code"] = resp.StatusCode

	return json.Marshal(result)
}
