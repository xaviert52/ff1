package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"flows/internal/domain"
	"flows/internal/dto"
	"flows/internal/service"

	"github.com/gin-gonic/gin"
)

type FlowHandler struct {
	FlowManager *service.FlowManager
}

func NewFlowHandler(manager *service.FlowManager) *FlowHandler {
	return &FlowHandler{FlowManager: manager}
}

// CreateFlow godoc
//
//	@Summary		Create a new Flow Definition
//	@Description	Define a new flow with its steps, schema, and transitions. The 'definition' field must follow the FlowDefinition structure.
//	@Tags			flow-management
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.CreateFlowRequest	true	"Flow Definition"
//	@Success		201		{object}	domain.Flow
//	@Failure		400		{object}	map[string]string
//	@Router			/flows [post]
func (h *FlowHandler) CreateFlow(c *gin.Context) {
	var req dto.CreateFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	defJSON, err := json.Marshal(req.Definition)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	flow := &domain.Flow{
		Name:        req.Name,
		Description: req.Description,
		Definition:  defJSON,
	}

	if err := h.FlowManager.DB.Create(flow).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, flow)
}

// StartFlow godoc
//
//	@Summary		Start a flow execution
//	@Description	Start a new instance of a flow. Returns the Execution ID (UUID) needed for subsequent steps.
//	@Tags			execution
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int						true	"Flow ID"
//	@Param			request	body		dto.StartFlowRequest	true	"Initial input parameters"
//	@Success		201		{object}	domain.Execution
//	@Failure		400		{object}	map[string]string
//	@Router			/flows/{id}/start [post]
func (h *FlowHandler) StartFlow(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid flow ID"})
		return
	}

	var req dto.StartFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	exec, err := h.FlowManager.StartFlow(uint(id), req.Input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, exec)
}

// GetCurrentStep godoc
//
//	@Summary		Get current step definition
//	@Description	Get the definition and JSON schema for the current step. Use this to render the form on the frontend.
//	@Tags			execution
//	@Produce		json
//	@Param			uuid	path		string	true	"Execution UUID"
//	@Success		200		{object}	domain.Step
//	@Failure		404		{object}	map[string]string
//	@Router			/executions/{uuid}/step [get]
func (h *FlowHandler) GetCurrentStep(c *gin.Context) {
	uuid := c.Param("uuid")
	step, _, err := h.FlowManager.GetCurrentStep(uuid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, step)
}

// SubmitStep godoc
//
//	@Summary		Submit step data
//	@Description	Validate and submit data for the current step. If successful, advances the flow to the next step.
//	@Tags			execution
//	@Accept			json
//	@Produce		json
//	@Param			uuid	path		string					true	"Execution UUID"
//	@Param			request	body		dto.SubmitStepRequest	true	"Form Data matching the step schema"
//	@Success		200		{object}	domain.Execution
//	@Failure		400		{object}	map[string]string
//	@Router			/executions/{uuid}/step [post]
func (h *FlowHandler) SubmitStep(c *gin.Context) {
	uuid := c.Param("uuid")

	rawData, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid body"})
		return
	}

	var req dto.SubmitStepRequest
	if err := json.Unmarshal(rawData, &req); err != nil {
		// Fallback: try to unmarshal directly as map (if user didn't wrap in "data" object, though they should)
		// Or better, check if rawData matches expected schema wrapper
		// Actually, dto.SubmitStepRequest expects { "data": { ... } }
		// But the error "validation error" comes from SubmitStep calling Validator.
		// If Unmarshal fails here, it returns "Invalid JSON format".
		// The user is getting a validation error, which means Unmarshal succeeded.
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}

	// If req.Data is nil (e.g. user sent { "input": ... } instead of { "data": ... }), handle it
	if req.Data == nil {
		// Try to parse rawData into a map to see if they sent fields directly or used wrong key
		var rawMap map[string]interface{}
		json.Unmarshal(rawData, &rawMap)
		if input, ok := rawMap["input"]; ok {
			// User sent "input" instead of "data", let's be flexible
			if inputMap, ok := input.(map[string]interface{}); ok {
				req.Data = inputMap
			}
		}
	}

	dataJSON, _ := json.Marshal(req.Data)
	exec, err := h.FlowManager.SubmitStep(uuid, dataJSON)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, exec)
}

// ListFlows godoc
//
//	@Summary		List all flows
//	@Description	Get a list of all available flow definitions
//	@Tags			flow-management
//	@Produce		json
//	@Success		200	{array}		domain.Flow
//	@Failure		500	{object}	map[string]string
//	@Router			/flows [get]
func (h *FlowHandler) ListFlows(c *gin.Context) {
	flows, err := h.FlowManager.ListFlows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, flows)
}

// GetExecution godoc
//
//	@Summary		Get execution details
//	@Description	Get full details of an execution including data and steps history
//	@Tags			execution
//	@Produce		json
//	@Param			uuid	path		string	true	"Execution UUID"
//	@Success		200		{object}	domain.Execution
//	@Failure		404		{object}	map[string]string
//	@Router			/executions/{uuid} [get]
func (h *FlowHandler) GetExecution(c *gin.Context) {
	uuid := c.Param("uuid")
	exec, err := h.FlowManager.GetExecutionDetails(uuid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, exec)
}

// RetryExecution godoc
//
//	@Summary		Retry a failed execution
//	@Description	Retry a failed execution from the current step, optionally updating input data
//	@Tags			execution
//	@Accept			json
//	@Produce		json
//	@Param			uuid	path		string					true	"Execution UUID"
//	@Param			request	body		map[string]interface{}	false	"New Input Data to merge"
//	@Success		200		{object}	domain.Execution
//	@Failure		400		{object}	map[string]string
//	@Router			/executions/{uuid}/retry [post]
func (h *FlowHandler) RetryExecution(c *gin.Context) {
	uuid := c.Param("uuid")

	var input map[string]interface{}
	// Optional body
	if c.Request.Body != http.NoBody {
		// Let's use a generic map for flexibility or reuse StartFlowRequest.Input
		// But here we might want to update specific fields.
		// Let's check if they sent { "input": ... } like StartFlow
		var body struct {
			Input map[string]interface{} `json:"input"`
		}
		if err := c.ShouldBindJSON(&body); err == nil {
			input = body.Input
		}
	}

	exec, err := h.FlowManager.RetryExecution(uuid, input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, exec)
}
