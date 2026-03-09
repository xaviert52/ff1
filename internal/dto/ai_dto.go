package dto

import "flows/internal/domain"

type GenerateFlowRequest struct {
	Prompt string `json:"prompt" binding:"required" example:"Create a flow that asks for email and sends a welcome message using the REST connector"`
}

type GenerateFlowResponse struct {
	Flow        *domain.Flow       `json:"flow,omitempty"`
	Connectors  []domain.Connector `json:"connectors,omitempty"`
	MissingInfo []string           `json:"missing_info,omitempty"`
	Explanation string             `json:"explanation,omitempty"`
}
