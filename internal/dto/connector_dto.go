package dto

import (
	"flows/internal/domain"
)

// CreateConnectorRequest payload for registering a new external service
type CreateConnectorRequest struct {
	Name     string               `json:"name" binding:"required" example:"PDF Generator Service"`
	Type     domain.ConnectorType `json:"type" binding:"required" example:"REST"`
	BaseURL  string               `json:"base_url" example:"https://api.pdf-service.com/v1"`
	AuthType domain.AuthType      `json:"auth_type" binding:"required" example:"API_KEY"`

	// Schema defines the expected input parameters for this connector using JSON Schema
	// This tells the flow designer what data is needed to call this service (e.g., template_id, user_data)
	Schema map[string]interface{} `json:"schema" swaggertype:"object,string" example:"type:object,properties:{template_id:{type:string},user_data:{type:object}},required:[template_id]"`

	// Policy defines resilience settings: timeouts, retries, etc.
	Policy ConnectorPolicy `json:"policy"`
}

// ConnectorPolicy defines the structure expected in the 'policy' field
type ConnectorPolicy struct {
	TimeoutMs      int `json:"timeout_ms" example:"5000"`
	MaxRetries     int `json:"max_retries" example:"3"`
	RetryBackoffMs int `json:"retry_backoff_ms" example:"1000"`
}

// CreateConnectorConfigRequest payload for environment-specific configuration (secrets)
type CreateConnectorConfigRequest struct {
	ConnectorID uint   `json:"connector_id" binding:"required" example:"1"`
	Environment string `json:"environment" binding:"required" example:"production"`

	// Config contains sensitive data like API keys or specific headers
	Config map[string]interface{} `json:"config" swaggertype:"object,string" example:"api_key:sk_live_123456,client_secret:abcde"`
}
