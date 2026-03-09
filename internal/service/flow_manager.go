package service

import (
	"encoding/json"
	"errors"
	"flows/internal/domain"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FlowManager struct {
	DB                 *gorm.DB
	StepValidator      *StepValidator
	SubscriptionClient *SubscriptionClient
	ConnectorService   *ConnectorService
}

func NewFlowManager(db *gorm.DB, validator *StepValidator, subscriptionClient *SubscriptionClient, connectorService *ConnectorService) *FlowManager {
	return &FlowManager{
		DB:                 db,
		StepValidator:      validator,
		SubscriptionClient: subscriptionClient,
		ConnectorService:   connectorService,
	}
}

// resolveTemplate replaces {{path.to.value}} with actual values from context
func resolveTemplate(template string, context map[string]interface{}) interface{} {
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)

	// If the template is EXACTLY one variable {{var}}, return the typed value (not string)
	if re.MatchString(template) {
		matches := re.FindStringSubmatch(template)
		if len(matches) == 2 && matches[0] == template {
			return getValueFromPath(matches[1], context)
		}
	}

	// Otherwise, perform string interpolation
	return re.ReplaceAllStringFunc(template, func(match string) string {
		path := strings.TrimSuffix(strings.TrimPrefix(match, "{{"), "}}")
		val := getValueFromPath(path, context)
		if val == nil {
			return ""
		}
		return fmt.Sprintf("%v", val)
	})
}

func getValueFromPath(path string, context map[string]interface{}) interface{} {
	parts := strings.Split(path, ".")
	var current interface{} = context

	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			if val, exists := m[part]; exists {
				current = val
			} else {
				return nil
			}
		} else if l, ok := current.([]interface{}); ok {
			// Array index support
			var idx int
			if _, err := fmt.Sscanf(part, "%d", &idx); err == nil {
				if idx >= 0 && idx < len(l) {
					current = l[idx]
				} else {
					return nil
				}
			} else {
				return nil
			}
		} else {
			return nil
		}
	}
	return current
}

// StartFlow initializes a new flow execution
func (m *FlowManager) StartFlow(flowID uint, input map[string]interface{}) (*domain.Execution, error) {
	var flow domain.Flow
	if err := m.DB.First(&flow, flowID).Error; err != nil {
		return nil, err
	}

	var definition domain.FlowDefinition
	if err := json.Unmarshal(flow.Definition, &definition); err != nil {
		return nil, fmt.Errorf("invalid flow definition: %w", err)
	}

	if definition.StartStep == "" {
		return nil, errors.New("flow definition has no start step")
	}

	inputJSON, _ := json.Marshal(input)
	execID := uuid.New().String()

	exec := &domain.Execution{
		ID:          execID,
		FlowID:      flowID,
		Status:      domain.StatusPending,
		CurrentStep: definition.StartStep,
		Data:        inputJSON,
		StepsData:   json.RawMessage("{}"),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := m.DB.Create(exec).Error; err != nil {
		return nil, err
	}

	if m.SubscriptionClient != nil {
		m.SubscriptionClient.SendStatus(exec)
	}

	// Auto-advance if start step is ACTION or DECISION
	startStep := definition.Steps[definition.StartStep]
	if startStep.Type == domain.StepTypeAction || startStep.Type == domain.StepTypeDecision {
		log.Printf("[FlowManager] Start step '%s' is automatic (%s). Auto-advancing...", startStep.ID, startStep.Type)
		// Trigger SubmitStep. Input is empty because mapping should pull from global Data.
		// We ignore the returned execution because we return the final state anyway.
		// However, if SubmitStep fails, we should return the error or the failed execution.
		advancedExec, err := m.SubmitStep(exec.ID, []byte("{}"))
		if err != nil {
			log.Printf("[FlowManager] Auto-advance failed: %v", err)
			// Return the execution (which should be in FAILED state) and the error
			return advancedExec, err // Or maybe just return advancedExec and let caller see status?
			// Standard practice: if it fails, return error.
		}
		return advancedExec, nil
	}

	return exec, nil
}

// GetCurrentStep returns the definition of the current step for an execution
func (m *FlowManager) GetCurrentStep(execID string) (*domain.Step, *domain.Execution, error) {
	var exec domain.Execution
	if err := m.DB.First(&exec, "id = ?", execID).Error; err != nil {
		return nil, nil, err
	}

	if exec.Status == domain.StatusCompleted || exec.Status == domain.StatusFailed {
		return nil, &exec, errors.New("execution is already finished")
	}

	var flow domain.Flow
	if err := m.DB.First(&flow, exec.FlowID).Error; err != nil {
		return nil, nil, err
	}

	var definition domain.FlowDefinition
	if err := json.Unmarshal(flow.Definition, &definition); err != nil {
		return nil, nil, err
	}

	step, ok := definition.Steps[exec.CurrentStep]
	if !ok {
		return nil, nil, fmt.Errorf("step %s not found in definition", exec.CurrentStep)
	}

	return &step, &exec, nil
}

// ListFlows returns all flow definitions
func (m *FlowManager) ListFlows() ([]domain.Flow, error) {
	var flows []domain.Flow
	if err := m.DB.Find(&flows).Error; err != nil {
		return nil, err
	}
	return flows, nil
}

// RetryExecution attempts to retry a failed execution from the current step
func (m *FlowManager) RetryExecution(execID string, newData map[string]interface{}) (*domain.Execution, error) {
	var exec domain.Execution
	if err := m.DB.First(&exec, "id = ?", execID).Error; err != nil {
		return nil, err
	}

	if exec.Status != domain.StatusFailed {
		return &exec, errors.New("only failed executions can be retried")
	}

	// Update global data if provided
	if len(newData) > 0 {
		// Merge with existing data
		var currentData map[string]interface{}
		json.Unmarshal(exec.Data, &currentData)
		if currentData == nil {
			currentData = make(map[string]interface{})
		}
		for k, v := range newData {
			currentData[k] = v
		}
		dataJSON, _ := json.Marshal(currentData)
		exec.Data = dataJSON
	}

	// Reset status to Running
	exec.Status = domain.StatusRunning
	exec.UpdatedAt = time.Now()
	if err := m.DB.Save(&exec).Error; err != nil {
		return nil, err
	}

	if m.SubscriptionClient != nil {
		m.SubscriptionClient.SendStatus(&exec)
	}

	// Trigger execution of the current step
	// We pass empty input because for ACTION steps (usually where it fails),
	// input is derived from mapping. If it was a FORM step, user should have used SubmitStep.
	// But if retry is for a FORM step that failed validation internally (unlikely to persist as FAILED),
	// or an ACTION step.
	// We assume retry is mostly for ACTION steps.
	return m.SubmitStep(exec.ID, []byte("{}"))
}

// GetExecutionDetails returns full details of an execution
func (m *FlowManager) GetExecutionDetails(execID string) (*domain.Execution, error) {
	var exec domain.Execution
	if err := m.DB.First(&exec, "id = ?", execID).Error; err != nil {
		return nil, err
	}
	return &exec, nil
}

// SubmitStep processes input for the current step and advances the flow
func (m *FlowManager) SubmitStep(execID string, input json.RawMessage) (*domain.Execution, error) {
	// Loop to handle consecutive automatic steps
	for {
		log.Printf("[FlowManager] Processing loop for execution %s", execID)
		step, exec, err := m.GetCurrentStep(execID)
		if err != nil {
			log.Printf("[FlowManager] Error getting current step: %v", err)
			return nil, err
		}
		log.Printf("[FlowManager] Current step: %s (Type: %s)", step.ID, step.Type)

		// 0. Resolve Input Mapping
		if len(step.InputMapping) > 0 {
			log.Printf("[FlowManager] Resolving Input Mapping for step %s...", step.ID)
			// Prepare context for mapping (global input + previous steps data)
			mappingContext := make(map[string]interface{})

			// Load global input
			var globalInput map[string]interface{}
			json.Unmarshal(exec.Data, &globalInput)
			mappingContext["global"] = globalInput

			// Load steps data
			var stepsData map[string]interface{}
			json.Unmarshal(exec.StepsData, &stepsData)
			mappingContext["steps"] = stepsData

			// Load current input (if any)
			var currentInputMap map[string]interface{}
			json.Unmarshal(input, &currentInputMap)
			mappingContext["input"] = currentInputMap

			mappedInput := make(map[string]interface{})
			for k, v := range step.InputMapping {
				// Handle both string templates and direct values
				if strVal, ok := v.(string); ok {
					val := resolveTemplate(strVal, mappingContext)
					mappedInput[k] = val
					log.Printf("[FlowManager] Mapped '%s' -> '%v'", k, val)
				} else {
					mappedInput[k] = v
					log.Printf("[FlowManager] Mapped (direct) '%s' -> '%v'", k, v)
				}
			}

			// Reassign input!
			mappedBytes, _ := json.Marshal(mappedInput)
			input = mappedBytes
			log.Printf("[FlowManager] New Input: %s", string(input))
		} else {
			log.Printf("[FlowManager] No Input Mapping defined. Using raw input.")
		}

		// 1. Process Logic based on Step Type
		var outputData json.RawMessage

		switch step.Type {
		case domain.StepTypeForm:
			log.Printf("[FlowManager] Validating FORM input...")
			// Validate User Input
			// Convert map schema to json.RawMessage for validator
			schemaJSON, _ := json.Marshal(step.Schema)
			if err := m.StepValidator.Validate(schemaJSON, input); err != nil {
				log.Printf("[FlowManager] Validation failed: %v", err)
				return nil, fmt.Errorf("validation error: %w", err)
			}
			outputData = input
			log.Printf("[FlowManager] Form Validated.")

		case domain.StepTypeAction:
			// Execute Connector
			if step.ConnectorID == 0 {
				return nil, errors.New("action step missing connector_id")
			}

			log.Printf("[FlowManager] Executing Connector ID %d...", step.ConnectorID)
			result, err := m.ConnectorService.ExecuteConnector(step.ConnectorID, input, "development", step.Config)
			if err != nil {
				log.Printf("[FlowManager] Connector execution failed: %v", err)
				exec.Status = domain.StatusFailed
				m.DB.Save(exec)
				if m.SubscriptionClient != nil {
					m.SubscriptionClient.SendStatus(exec)
				}
				return nil, fmt.Errorf("connector failed: %w", err)
			}
			log.Printf("[FlowManager] Connector success. Result: %s", string(result))
			outputData = result

		case domain.StepTypeDecision:
			// Decision logic usually doesn't produce output, but evaluates transitions
			outputData = input
			log.Printf("[FlowManager] Decision step processed.")
		}

		// 2. Update Execution Data
		var currentStepsData map[string]interface{}
		json.Unmarshal(exec.StepsData, &currentStepsData)
		if currentStepsData == nil {
			currentStepsData = make(map[string]interface{})
		}

		var outputMap interface{}
		json.Unmarshal(outputData, &outputMap)
		currentStepsData[exec.CurrentStep] = outputMap

		newStepsData, _ := json.Marshal(currentStepsData)
		exec.StepsData = newStepsData

		// 3. Determine Next Step
		nextStepID := step.NextStep

		// Check conditional transitions
		if len(step.Transitions) > 0 {
			log.Printf("[FlowManager] Evaluating transitions...")

			// Re-create mapping context if needed (it was local to InputMapping block)
			// Or better, define it outside. For now, let's just recreate it quickly.
			transitionContext := make(map[string]interface{})

			var globalInput map[string]interface{}
			json.Unmarshal(exec.Data, &globalInput)
			transitionContext["global"] = globalInput

			var stepsData map[string]interface{}
			json.Unmarshal(exec.StepsData, &stepsData)
			transitionContext["steps"] = stepsData

			// Include current output as "input" for next steps or condition?
			// Usually conditions check steps data.
			// Let's add current outputData as "output"
			var currentOutput map[string]interface{}
			json.Unmarshal(outputData, &currentOutput)
			transitionContext["output"] = currentOutput

			for _, t := range step.Transitions {
				// Evaluate condition
				// Resolve variables in condition string
				resolvedCondition := resolveTemplate(t.Condition, transitionContext)

				// Very basic evaluation: check if string is "true" or matches equality
				// Since we don't have an expression engine yet, let's implement basic "A == B"
				conditionStr := fmt.Sprintf("%v", resolvedCondition)
				log.Printf("[FlowManager] Checking condition: %s (resolved: %s)", t.Condition, conditionStr)

				// Handle "true" literal
				if conditionStr == "true" {
					nextStepID = t.NextStep
					log.Printf("[FlowManager] Condition matched (true)! Next step: %s", nextStepID)
					break
				}

				// Handle "A == B"
				if strings.Contains(conditionStr, "==") {
					parts := strings.Split(conditionStr, "==")
					if len(parts) == 2 {
						left := strings.TrimSpace(parts[0])
						right := strings.TrimSpace(parts[1])
						if left == right {
							nextStepID = t.NextStep
							log.Printf("[FlowManager] Condition matched (%s == %s)! Next step: %s", left, right, nextStepID)
							break
						}
					}
				}
			}
		}

		log.Printf("[FlowManager] Moving to next step: '%s'", nextStepID)

		// 4. Advance
		if nextStepID == "" {
			exec.Status = domain.StatusCompleted
			exec.CurrentStep = ""
			exec.UpdatedAt = time.Now()
			m.DB.Save(exec)
			if m.SubscriptionClient != nil {
				m.SubscriptionClient.SendStatus(exec)
			}
			log.Printf("[FlowManager] Execution COMPLETED.")
			return exec, nil // Finish
		} else {
			exec.CurrentStep = nextStepID
			exec.Status = domain.StatusRunning
			exec.UpdatedAt = time.Now()
			m.DB.Save(exec)

			if m.SubscriptionClient != nil {
				m.SubscriptionClient.SendStatus(exec)
			}

			// Check next step type
			nextStepDef, _, err := m.GetCurrentStep(exec.ID)
			if err != nil {
				return nil, err
			}

			if nextStepDef.Type == domain.StepTypeForm {
				exec.Status = domain.StatusWaiting
				exec.UpdatedAt = time.Now()
				m.DB.Save(exec)
				if m.SubscriptionClient != nil {
					m.SubscriptionClient.SendStatus(exec)
				}
				log.Printf("[FlowManager] Execution WAITING for user input (Form).")
				return exec, nil // Return to user
			}

			// Prepare input for next step
			input = outputData
		}
	}
}
