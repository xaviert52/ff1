package domain

import (
	"encoding/json"
	"time"
)

type ConnectorType string
type AuthType string

const (
	ConnectorTypeREST ConnectorType = "REST"
	ConnectorTypeSOAP ConnectorType = "SOAP"

	AuthTypeNone   AuthType = "NONE"
	AuthTypeAPIKey AuthType = "API_KEY"
	AuthTypeOAuth2 AuthType = "OAUTH2"
)

type Connector struct {
	ID        uint            `json:"id" gorm:"primaryKey"`
	Name      string          `json:"name" gorm:"not null"`
	Type      ConnectorType   `json:"type" gorm:"not null"`
	BaseURL   string          `json:"base_url"`
	AuthType  AuthType        `json:"auth_type" gorm:"not null"`
	Schema    json.RawMessage `json:"schema" gorm:"type:json" swaggertype:"object,string" example:"type:object,properties:{name:{type:string}},required:[name]"`
	Policy    json.RawMessage `json:"policy" gorm:"type:json" swaggertype:"object,string" example:"timeout_ms:5000,max_retries:3"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type ConnectorConfig struct {
	ID          uint            `json:"id" gorm:"primaryKey"`
	ConnectorID uint            `json:"connector_id" gorm:"not null;uniqueIndex:idx_connector_env"`
	Environment string          `json:"environment" gorm:"not null;uniqueIndex:idx_connector_env"`
	Config      json.RawMessage `json:"config" gorm:"type:json" swaggertype:"object,string" example:"api_key:secret_key"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type ConnectorRepository interface {
	Create(c *Connector) error
	GetByID(id uint) (*Connector, error)
	Update(c *Connector) error
	Delete(id uint) error
	List() ([]Connector, error)

	SaveConfig(c *ConnectorConfig) error
	GetConfig(connectorID uint, env string) (*ConnectorConfig, error)
}
