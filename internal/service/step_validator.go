package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

type StepValidator struct{}

func NewStepValidator() *StepValidator {
	return &StepValidator{}
}

func (v *StepValidator) Validate(schemaJSON json.RawMessage, dataJSON json.RawMessage) error {
	// Cambia esta condición para manejar nulos y vacíos correctamente
	if len(schemaJSON) == 0 || string(schemaJSON) == "null" || string(schemaJSON) == "" {
		return nil // Si no hay esquema o es null, no se requiere validación
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("schema.json", strings.NewReader(string(schemaJSON))); err != nil {
		return fmt.Errorf("invalid schema: %w", err)
	}

	schema, err := compiler.Compile("schema.json")
	if err != nil {
		return fmt.Errorf("failed to compile schema: %w", err)
	}

	var data interface{}
	if err := json.Unmarshal(dataJSON, &data); err != nil {
		return fmt.Errorf("invalid data format: %w", err)
	}

	if err := schema.Validate(data); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}
