package aihelpers

import (
	"context"
	"errors"
	"fmt"
	"github.com/openai/openai-go/option"

	openai "github.com/openai/openai-go"
)

type AIClient struct {
	APIKey string
	Model  string
	Client *openai.Client
}

// NewOpenAIClient initializes a new OpenAI client with the API key and model.
func NewOpenAIClient(apiKey, model string) *AIClient {
	return &AIClient{
		APIKey: apiKey,
		Model:  model,
		Client: openai.NewClient(
			option.WithAPIKey(apiKey), // defaults to os.LookupEnv("OPENAI_API_KEY")
		),
	}
}

// SetModel sets the model to be used for OpenAI requests.
func (client *AIClient) SetModel(model string) {
	client.Model = model
}

// PromptRequest represents the parameters for a prompt request.
type PromptRequest struct {
	Prompt      string
	MaxTokens   int
	Temperature float64
}

// TODO: Convert to a streaming version to avoid large memory usage on large files
// Prompt sends a prompt to OpenAI and returns the response text.
func (client *AIClient) Prompt(ctx context.Context, req PromptRequest) (string, *openai.ChatCompletion, error) {
	if client.APIKey == "" {
		return "", nil, errors.New("API key is not set")
	}

	if client.Model == "" {
		return "", nil, errors.New("model is not set")
	}

	//TODO: Use req to tailor the request

	chatCompletion, err := client.Client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(req.Prompt),
		}),
		Model: openai.F(client.Model),
	})
	if err != nil {
		// wrap the error with additional context
		return "", nil, fmt.Errorf("failed to create chat completion: %w", err)
	}

	if len(chatCompletion.Choices) == 0 {
		return "", nil, errors.New("no response choices from OpenAI")
	}

	return chatCompletion.Choices[0].Message.Content, chatCompletion, nil
}
