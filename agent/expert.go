package agent

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/genai"
)

// Expert represent a chat with a business expert.
type Expert struct {
	Name        string                       `json:"name"`
	Description string                       `json:"description"`
	ModelName   string                       `json:"model_name"`
	Config      *genai.GenerateContentConfig `json:"config"`
	Library     Library
	chat        *genai.Chat
}

func NewExpert(name, description string) *Expert {
	return &Expert{
		Name:        name,
		Description: description,
	}
}

func (e *Expert) Start(ctx context.Context, client *genai.Client) error {
	chat, err := client.Chats.Create(ctx, e.ModelName, e.Config, nil)
	if err != nil {
		return err
	}
	e.chat = chat
	return nil
}

// Ask is a simple wrapper on top of Chat.Send to make it simpler for callbacks.
func (e *Expert) Ask(ctx context.Context, parts ...*genai.Part) (*genai.Content, error) {
	resp, err := e.chat.Send(ctx, parts...)
	if err != nil {
		return nil, err
	}
	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no response from expert %s", e.Name)
	}
	part0 := resp.Candidates[0].Content.Parts[0]
	if part0.FunctionCall != nil {
		if e.Library == nil {
			return nil, fmt.Errorf("expert %s doesn't know how to make function calls", e.Name)
		}

		// Make the callback. No possible error, this error should be sent via 'resp'
		resp := e.Library(ctx, part0.FunctionCall)

		// Ask again the expert with the response he asked for
		// until we have a real response.
		return e.Ask(ctx, &genai.Part{FunctionResponse: resp})
	}
	return resp.Candidates[0].Content, nil
}

// Declaration returns the function declaration to ask this expert.
func (e *Expert) Declaration() *genai.FunctionDeclaration {
	return &genai.FunctionDeclaration{
		Name:        e.Name,
		Description: e.Description,
		Parameters: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"question": {
					Type:        genai.TypeString,
					Description: "The question to ask the expert.",
				},
			},
			Required: []string{"question"},
		},
		Response: &genai.Schema{
			Type:        genai.TypeString,
			Description: "Expert's reponse.",
		},
	}
}

// Call perform the call of asking this expert.
func (e *Expert) Call(ctx context.Context, id string, args map[string]any) *genai.FunctionResponse {
	d := e.Declaration()
	fresp := &genai.FunctionResponse{
		ID:   id,
		Name: d.Name,
	}

	arg0 := args[d.Parameters.Required[0]]
	question, ok := arg0.(string)
	if !ok {
		fresp.Response["error"] = fmt.Errorf("invalid type got %T, expected string", arg0)
		return fresp
	}

	response, err := e.Ask(ctx, &genai.Part{Text: question})
	if err != nil {
		fresp.Response["error"] = fmt.Errorf("something went wrong while calling the expert: %w", err)
		return fresp
	}

	r := response.Parts[0].Text
	log.Printf("Expert %q: \n        %q\n        %q", e.Name, question, r)
	fresp.Response = map[string]any{
		"output": r,
	}
	return fresp
}
