package service

import (
	"context"
	"encoding/json"
	"flows/internal/domain"
	"flows/internal/dto"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

type AIService struct {
	Client *openai.Client
}

func NewAIService(apiKey string) *AIService {
	return &AIService{
		Client: openai.NewClient(apiKey),
	}
}

func (s *AIService) GenerateFlow(prompt string) (*dto.GenerateFlowResponse, error) {
	systemPrompt := `You are an expert Flow Engineer for a workflow automation system.
Your task is to convert natural language descriptions into a JSON structure representing a Flow and optionally Connectors.

The system has the following Domain Models:

1. Flow Definition (JSON):
{
  "start_step": "step_id_1",
  "steps": {
    "step_id_1": {
      "id": "step_id_1",
      "type": "FORM" | "ACTION" | "DECISION",
      "description": "...",
      "next_step": "step_id_2",
      "schema": { ... JSON Schema for FORM input ... },
      "connector_id": 123 (integer, placeholder if new),
      "config": { ... key-value pairs ... }
    }
  }
}

2. Connector (if needed):
{
  "name": "...",
  "type": "REST" | "SOAP",
  "base_url": "...",
  "auth_type": "NONE" | "API_KEY" | "OAUTH2"
}

Output Format (JSON Only):
{
  "flow": {
     "name": "...",
     "description": "...",
     "definition": { ... FlowDefinition object ... }
  },
  "connectors": [ ... list of connectors to create ... ],
  "missing_info": [ ... list of questions if info is missing ... ],
  "explanation": "Brief explanation of what was generated"
}

Rules:
- If the user mentions an external API (e.g. "send email"), create a Connector definition for it.
- If the user implies a form (e.g. "ask for name"), create a FORM step with a JSON Schema.
- Use logical IDs for steps (e.g., "ask_name", "send_email").
- If critical info is missing (e.g., "what is the API URL?"), add it to "missing_info".
- Return valid JSON only.`

	resp, err := s.Client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4o,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			ResponseFormat: &openai.ChatCompletionResponseFormat{
				Type: openai.ChatCompletionResponseFormatTypeJSONObject,
			},
		},
	)

	if err != nil {
		return nil, fmt.Errorf("AI generation failed: %w", err)
	}

	var result struct {
		Flow        *domain.Flow       `json:"flow"`
		Connectors  []domain.Connector `json:"connectors"`
		MissingInfo []string           `json:"missing_info"`
		Explanation string             `json:"explanation"`
	}

	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &result); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	return &dto.GenerateFlowResponse{
		Flow:        result.Flow,
		Connectors:  result.Connectors,
		MissingInfo: result.MissingInfo,
		Explanation: result.Explanation,
	}, nil
}
