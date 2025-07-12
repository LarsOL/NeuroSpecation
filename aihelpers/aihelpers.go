package aihelpers

import (
	"context"
	"errors"
	"fmt"
	"io"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type AIClient struct {
	APIKey string
	Model  string
	Client *openai.Client
}

// NewOpenAIClient initializes a new OpenAI client with the API key and model.
func NewOpenAIClient(apiKey, model string, opts ...option.RequestOption) *AIClient {
	opts = append(opts, option.WithAPIKey(apiKey))
	return &AIClient{
		APIKey: apiKey,
		Model:  model,
		Client: openai.NewClient(opts...),
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

func (client *AIClient) PromptStream(ctx context.Context, req PromptRequest) (string, error) {
	if client.APIKey == "" {
		return "", errors.New("API key is not set")
	}

	if client.Model == "" {
		return "", errors.New("model is not set")
	}

	// Use req to tailor the request
	stream := client.Client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(req.Prompt),
		}),
		Model: openai.F(client.Model),
	})

	var responseContent string
	for stream.Next() {
		chatCompletion := stream.Current()
		responseContent += chatCompletion.Choices[0].Delta.Content
	}

	if stream.Err() != nil && !errors.Is(stream.Err(), io.EOF) {
		return "", fmt.Errorf("error receiving stream response: %w", stream.Err())
	}

	if responseContent == "" {
		return "", errors.New("no response content from OpenAI")
	}

	return responseContent, nil
}
