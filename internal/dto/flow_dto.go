package dto

import (
	"flows/internal/domain"
)

// CreateFlowRequest is the payload for creating a new Flow definition
type CreateFlowRequest struct {
	Name        string                `json:"name" binding:"required" example:"Onboarding Process"`
	Description string                `json:"description" example:"Employee onboarding flow with document verification"`
	Definition  domain.FlowDefinition `json:"definition" binding:"required"`
}

// StartFlowRequest payload to initiate a flow instance
type StartFlowRequest struct {
	Input map[string]interface{} `json:"input" swaggertype:"object,string" example:"referrer:web_portal,campaign_id:summer_2024"`
}

// SubmitStepRequest payload for submitting form data
type SubmitStepRequest struct {
	Data map[string]interface{} `json:"data" binding:"required" swaggertype:"object,string" example:"name:John Doe,age:30,email:john@example.com"`
}
