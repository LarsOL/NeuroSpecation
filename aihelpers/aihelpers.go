package aihelpers

import (
	"context"
	"errors"
	"fmt"
	"github.com/openai/openai-go/option"

	openai "github.com/openai/openai-go"
)

type AIClient struct {
	APIKey   string
	Model    string
	Client   *openai.Client
	LocalLLM bool
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

// NewLocalLLMClient initializes a new client for a local LLM using Ollama.
func NewLocalLLMClient() *AIClient {
	return &AIClient{
		LocalLLM: true,
	}
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

func (client *AIClient) Prompt(ctx context.Context, req PromptRequest) (string, error) {
	if client.LocalLLM {
		return client.promptLocalLLM(req)
	}

	if client.APIKey == "" {
		return "", errors.New("API key is not set")
	}

	if client.Model == "" {
		return "", errors.New("model is not set")
	}

	chatCompletion, err := client.Client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(req.Prompt),
		}),
		Model: openai.F(client.Model),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create chat completion: %w", err)
	}

	if len(chatCompletion.Choices) == 0 {
		return "", errors.New("no response choices from OpenAI")
	}

	return chatCompletion.Choices[0].Message.Content, nil
}

import (
	"context"
	"fmt"
	"github.com/ollama/ollama/api"
)

func (client *AIClient) promptLocalLLM(req PromptRequest) (string, error) {
	ollamaClient, err := api.ClientFromEnvironment()
	if err != nil {
		return "", fmt.Errorf("failed to create Ollama client: %w", err)
	}

	messages := []api.Message{
		{
			Role:    "system",
			Content: "Provide very brief, concise responses",
		},
		{
			Role:    "user",
			Content: req.Prompt,
		},
	}

	ctx := context.Background()
	chatReq := &api.ChatRequest{
		Model:    "llama3.2",
		Messages: messages,
	}

	var responseContent string
	respFunc := func(resp api.ChatResponse) error {
		responseContent += resp.Message.Content
		return nil
	}

	err = ollamaClient.Chat(ctx, chatReq, respFunc)
	if err != nil {
		return "", fmt.Errorf("failed to get response from local LLM: %w", err)
	}

	return responseContent, nil
}
