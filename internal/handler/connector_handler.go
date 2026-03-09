package handler

import (
	"encoding/json"
	"flows/internal/domain"
	"flows/internal/dto"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ConnectorHandler struct {
	DB *gorm.DB
}

func NewConnectorHandler(db *gorm.DB) *ConnectorHandler {
	return &ConnectorHandler{DB: db}
}

// CreateConnector godoc
//
//	@Summary		Create a new connector
//	@Description	Register a new external service. The 'schema' field defines the input contract, and 'policy' defines resilience settings.
//	@Tags			connectors
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.CreateConnectorRequest	true	"Connector definition"
//	@Success		201		{object}	domain.Connector
//	@Failure		400		{object}	map[string]string
//	@Router			/connectors [post]
func (h *ConnectorHandler) CreateConnector(c *gin.Context) {
	var req dto.CreateConnectorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	schemaJSON, _ := json.Marshal(req.Schema)
	policyJSON, _ := json.Marshal(req.Policy)

	connector := &domain.Connector{
		Name:     req.Name,
		Type:     req.Type,
		BaseURL:  req.BaseURL,
		AuthType: req.AuthType,
		Schema:   schemaJSON,
		Policy:   policyJSON,
	}

	if err := h.DB.Create(connector).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, connector)
}

// ListConnectors godoc
//
//	@Summary		List all connectors
//	@Description	Get a list of all registered connectors
//	@Tags			connectors
//	@Produce		json
//	@Success		200	{array}		domain.Connector
//	@Failure		500	{object}	map[string]string
//	@Router			/connectors [get]
func (h *ConnectorHandler) ListConnectors(c *gin.Context) {
	var connectors []domain.Connector
	if err := h.DB.Find(&connectors).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, connectors)
}

// CreateConfig godoc
//
//	@Summary		Create connector configuration
//	@Description	Configure environment-specific secrets (e.g., API keys) for a connector.
//	@Tags			connectors
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.CreateConnectorConfigRequest	true	"Connector Secrets"
//	@Success		201		{object}	domain.ConnectorConfig
//	@Failure		400		{object}	map[string]string
//	@Router			/connectors/config [post]
func (h *ConnectorHandler) CreateConfig(c *gin.Context) {
	var req dto.CreateConnectorConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	configJSON, _ := json.Marshal(req.Config)

	config := &domain.ConnectorConfig{
		ConnectorID: req.ConnectorID,
		Environment: req.Environment,
		Config:      configJSON,
	}

	if err := h.DB.Create(config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, config)
}
