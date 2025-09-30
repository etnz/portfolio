package agent

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/etnz/portfolio"
	"github.com/etnz/portfolio/docs"
	"github.com/etnz/portfolio/renderer"
	"google.golang.org/genai"
)

const model = "gemini-2.5-pro"

// creates the facilitator
func newFacilitator(experts ...*Expert) *Expert {
	return &Expert{
		Name: "Facilitator",
		// Used by facilitators to know what they can expected from the expert
		Description: ``,
		ModelName:   model,
		Config: &genai.GenerateContentConfig{
			Tools: []*genai.Tool{
				{FunctionDeclarations: NewDeclaration(experts)},
			},
			SystemInstruction: &genai.Content{Parts: []*genai.Part{{Text: `
			As a facilitator you are in charge of the conversation and solving the user's request. 
			
			Learn about the expert's skill that you can get from the Tools to ask them questions.
			They are at your service and 100% dedicated to you, they keep context of your previous questions.

			Analyse sentiment of user request, he is here primarily to get news or information about his assets
			in his portfolio.
			If he is angry try to understand why, and seek for a clear user approval.
			
			Devise a plan of questions to ask to each experts and come up with the best reponse to the user's request.

			The user will assume that you know about his security tickers, checked the portfolio first to understand what they are.
		`}}},
		},
		Library: NewLibrary(experts),
	}
}

func NewTrader() *Expert {
	return &Expert{
		Name: "Trader",
		Description: `This is an expert trader, 
		Very well aware of all the financial products and institutions, 
		about the latest news about the different funds or companies. 
		Ask the Trader whenever you need recent or grounding information.`,
		ModelName: model,
		Config: &genai.GenerateContentConfig{
			Tools: []*genai.Tool{
				{GoogleSearch: &genai.GoogleSearch{}},
			},
			SystemInstruction: &genai.Content{Parts: []*genai.Part{{Text: `
			You are a expert in Trading, you can search and find about anything related to 
			financial institutions, companies, markets, funds etc. You Leverage Google Search to
			ground your assertions in a solid truth.
			You can get the latests news too, and you know how to relate them to the user's request.
				`}}},
		},
	}
}

func NewAccountant() *Expert {

	lib := []Function{Declarations}

	return &Expert{
		Name: "Accountant",
		Description: `This is the Accountant. He is in charge of reading and editing the user's portfolio's ledger.
		He can perform many operations on the ledger to compute the relevant figure about the user's wealth.`,
		ModelName: model,
		Config: &genai.GenerateContentConfig{
			Tools: []*genai.Tool{
				{FunctionDeclarations: NewDeclaration(lib)},
			},
			SystemInstruction: &genai.Content{Parts: []*genai.Part{{Text: `
				You are an accountant in charge of the user's portfolio's ledger.
				You know how to use the Tools to extract relevant information about the user's portfolio and wealth.
				You are part of a team of experts, yours is everything about the user's portfolio. They might ask
				you questions about the user's portfolio, pardon their approximative language and figure out what they meant.

				Use the available tools to get for information about the user's portfolio 
				  - list of held securities
				  - positions
				  - holdings
			`}}},
		},
		Library: NewLibrary(lib),
	}
}

// Func implements a simple Function
type Func struct {
	// Declare this function
	Decl *genai.FunctionDeclaration
	// Call this function
	Func func(ctx context.Context, id string, args map[string]any) *genai.FunctionResponse
}

func (f *Func) Declaration() *genai.FunctionDeclaration { return f.Decl }
func (f *Func) Call(ctx context.Context, id string, args map[string]any) *genai.FunctionResponse {
	return f.Func(ctx, id, args)
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// The following implementation is not scalable, it will do it for the MVP not further.

var Declarations = &Func{

	Decl: &genai.FunctionDeclaration{
		Name: "Declarations",
		Description: `Declarations list all securities in this portfolio, and whether or not they are held on the given day.

		It details the user's choosen ticker, the security ID and the usual name for this security.
		`,
		Parameters: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"date": {
					Type: genai.TypeString,
					Description: `The date on which to compute the holdings. Today is the default.
					Otherwise it uses a flexible date format based on YYYY-MM-DD:
					
					` + must(docs.GetTopic("dates")),
				},
			},
			// Required: []string{"date"},
		},
		Response: &genai.Schema{
			Type:        genai.TypeString,
			Description: "A markdown-formatted table of all the known securities in the portfolio, with their user preferred ticker their full ID and a longer description.",
		},
	},
	Func: func(ctx context.Context, id string, args map[string]any) *genai.FunctionResponse {

		date, err := parseDate(args)
		if err != nil {
			return &genai.FunctionResponse{
				ID:   id,
				Name: "Declarations",
				Response: map[string]any{
					"error": err.Error(),
				},
			}
		}

		pos, err := declarations(date)
		if err != nil {
			return &genai.FunctionResponse{
				ID:   id,
				Name: "Declarations",
				Response: map[string]any{
					"error": err.Error(),
				},
			}
		}

		// For now, we're just returning a placeholder string.
		// In a real implementation, you'd capture the output of `cmd.Execute`
		// which prints to stdout and return that.
		return &genai.FunctionResponse{
			ID:   id,
			Name: "Positions",
			Response: map[string]any{
				"output": pos,
			},
		}
	},
}

// private implementation for to render the declarations.
func declarations(d portfolio.Date) (string, error) {
	ledger, err := DecodeLedger()
	if err != nil {
		return "", fmt.Errorf("could not load ledger: %w", err)
	}
	s := ledger.NewSnapshot(d)
	return renderer.DeclarationMarkdown(s), nil
}

// DecodeLedger decodes the ledger from the application's default ledger file.
// If the file does not exist, it returns a new empty ledger.
func DecodeLedger() (*portfolio.Ledger, error) {
	ledgerFile := "transactions.jsonl"
	// temp
	f, err := os.Open(ledgerFile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// If the file doesn't exist, it's an empty ledger.
			return portfolio.NewLedger(), nil
		}
		return nil, fmt.Errorf("could not open ledger file %q: %w", ledgerFile, err)
	}
	defer f.Close()

	ledger, err := portfolio.DecodeLedger(f)
	if err != nil {
		return nil, fmt.Errorf("could not decode ledger file %q: %w", ledgerFile, err)
	}
	return ledger, nil
}

func parseDate(args map[string]any) (portfolio.Date, error) {
	idate, hasDate := args["date"]
	if !hasDate {
		return portfolio.Today(), nil
	}
	sdate, ok := idate.(string)
	if !ok {
		return portfolio.Today(), fmt.Errorf("argument 'date' is not a string as expected but %T", idate)
	}

	date, err := portfolio.ParseDate(sdate)
	if err != nil {
		return portfolio.Today(), fmt.Errorf("argument 'date' must be a valid date got %q. Below is the doc about the format date\n\n%s ", sdate, must(docs.GetTopic("dates")))
	}

	return date, nil
}
