package agent

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

type Library func(context.Context, *genai.FunctionCall) *genai.FunctionResponse

type Function interface {
	// Declare this function
	Declaration() *genai.FunctionDeclaration
	// Call this function
	Call(ctx context.Context, id string, args map[string]any) *genai.FunctionResponse
}

// ExpertsCallback call an expert.
func NewLibrary[T Function](functions []T) Library {
	return func(ctx context.Context, call *genai.FunctionCall) *genai.FunctionResponse {
		for _, e := range functions {
			d := e.Declaration()
			if d.Name == call.Name {
				return e.Call(ctx, call.ID, call.Args)
			}
		}
		return &genai.FunctionResponse{
			ID:   call.ID,
			Name: call.Name,
			Response: map[string]any{
				"error": fmt.Errorf("unknown function %s", call.Name),
			},
		}
	}
}

func NewDeclaration[T Function](functions []T) []*genai.FunctionDeclaration {
	result := make([]*genai.FunctionDeclaration, 0, len(functions))
	for _, e := range functions {
		result = append(result, e.Declaration())
	}
	return result
}
