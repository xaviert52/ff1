package domain

import (
	"encoding/json"
	"time"
)

// --- Enums ---

type StepType string

const (
	StepTypeForm     StepType = "FORM"     // Human input
	StepTypeAction   StepType = "ACTION"   // Automated task/connector
	StepTypeDecision StepType = "DECISION" // Conditional logic
)

type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "PENDING"
	StatusRunning   ExecutionStatus = "RUNNING"
	StatusWaiting   ExecutionStatus = "WAITING" // Waiting for user input
	StatusCompleted ExecutionStatus = "COMPLETED"
	StatusFailed    ExecutionStatus = "FAILED"
	StatusSuspended ExecutionStatus = "SUSPENDED" // Waiting for async callback
)

// --- Flow Definitions ---

type Flow struct {
	ID          uint            `json:"id" gorm:"primaryKey"`
	Name        string          `json:"name" gorm:"not null"`
	Description string          `json:"description"`
	Definition  json.RawMessage `json:"definition" gorm:"type:json" swaggertype:"object,string" example:"start_step:step1,steps:{step1:{id:step1,type:FORM,next_step:step2}}"` // Stores FlowDefinition
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// FlowDefinition represents the structure of the JSON stored in Flow.Definition
type FlowDefinition struct {
	StartStep string          `json:"start_step" example:"personal_info"`
	Steps     map[string]Step `json:"steps"`
}

type Step struct {
	ID           string                 `json:"id" example:"personal_info"`
	Type         StepType               `json:"type" example:"FORM"`
	Description  string                 `json:"description" example:"Collect personal details"`
	Schema       map[string]interface{} `json:"schema,omitempty" swaggertype:"object,string" example:"type:object,properties:{name:{type:string}},required:[name]"` // JSON Schema for FORM input
	ConnectorID  uint                   `json:"connector_id,omitempty" example:"1"`                                                                                 // For ACTION steps
	Config       map[string]interface{} `json:"config,omitempty" swaggertype:"object,string" example:"param1:value1"`                                               // Connector/Action config
	InputMapping map[string]interface{} `json:"input_mapping,omitempty" swaggertype:"object,string" example:"param1:{{steps.step1.data.param1}}"`                   // Field mapping
	NextStep     string                 `json:"next_step,omitempty" example:"next_step_id"`                                                                         // Simple transition
	Transitions  []Transition           `json:"transitions,omitempty"`                                                                                              // Conditional transitions
}

type Transition struct {
	Condition string `json:"condition" example:"age >= 18"` // Simple expression or key
	NextStep  string `json:"next_step" example:"adult_step"`
}

// --- Execution State ---

type Execution struct {
	ID          string          `json:"id" gorm:"primaryKey"` // UUID
	FlowID      uint            `json:"flow_id"`
	Status      ExecutionStatus `json:"status"`
	CurrentStep string          `json:"current_step"`
	Data        json.RawMessage `json:"data" gorm:"type:json" swaggertype:"object,string"`       // Accumulated data
	StepsData   json.RawMessage `json:"steps_data" gorm:"type:json" swaggertype:"object,string"` // Map[StepID]Output
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// --- Interfaces ---

type FlowRepository interface {
	Create(f *Flow) error
	GetByID(id uint) (*Flow, error)
	List() ([]Flow, error)

	CreateExecution(e *Execution) error
	GetExecution(id string) (*Execution, error)
	UpdateExecution(e *Execution) error
}
