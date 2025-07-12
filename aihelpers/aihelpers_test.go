package aihelpers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	openai "github.com/openai/openai-go"
)

func TestNewAIClient(t *testing.T) {
	client := NewOpenAIClient("test_api_key", "test_model")
	if client.APIKey != "test_api_key" {
		t.Errorf("Expected APIKey to be 'test_api_key', but got %s", client.APIKey)
	}
	if client.Model != "test_model" {
		t.Errorf("Expected Model to be 'test_model', but got %s", client.Model)
	}
	if client.Client == nil {
		t.Error("Expected Client to be initialized, but it was nil")
	}
}

func TestAIClient_SetModel(t *testing.T) {
	client := NewOpenAIClient("test_api_key", "test_model")
	client.SetModel("new_model")
	if client.Model != "new_model" {
		t.Errorf("Expected Model to be 'new_model', but got %s", client.Model)
	}
}

func TestAIClient_Prompt(t *testing.T) {
	// Create a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		//nolint:goconst
		response := `{ 
			"id": "chatcmpl-123", 
			"object": "chat.completion", 
			"created": 1677652288, 
			"model": "gpt-3.5-turbo-0613", 
			"choices": [{
				"index": 0, 
				"message": { 
					"role": "assistant", 
					"content": "Hello, how can I help you?" 
				}, 
				"finish_reason": "stop" 
			}], 
			"usage": { 
				"prompt_tokens": 9, 
				"completion_tokens": 12, 
				"total_tokens": 21 
			} 
		}`
		_, err := w.Write([]byte(response))
		if err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer mockServer.Close()

	// Create a new client with the mock server's URL
	client := NewOpenAIClient("test_api_key", "test_model")
	client.Client.BaseURL = mockServer.URL

	// Test case 1: Successful prompt
	req := PromptRequest{
		Prompt:      "Hello",
		MaxTokens:   50,
		Temperature: 0.7,
	}
	content, _, err := client.Prompt(context.Background(), req)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	if content != "Hello, how can I help you?" {
		t.Errorf("Expected content 'Hello, how can I help you?', but got '%s'", content)
	}

	// Test case 2: No API key
	client.APIKey = ""
	_, _, err = client.Prompt(context.Background(), req)
	if err == nil {
		t.Error("Expected an error for no API key, but got nil")
	}
	client.APIKey = "test_api_key" // Reset API key

	// Test case 3: No model
	client.Model = ""
	_, _, err = client.Prompt(context.Background(), req)
	if err == nil {
		t.Error("Expected an error for no model, but got nil")
	}
	client.Model = "test_model" // Reset model
}

func TestAIClient_PromptStream(t *testing.T) {
	// Create a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Simulate a streaming response
		data := []string{
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-3.5-turbo-0613","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-3.5-turbo-0613","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-3.5-turbo-0613","choices":[{"index":0,"delta":{"content":", "},"finish_reason":null}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-3.5-turbo-0613","choices":[{"index":0,"delta":{"content":"how can I help you?"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-3.5-turbo-0613","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		}
		for _, d := range data {
			_, err := w.Write([]byte("data: " + d + "\n\n"))
			if err != nil {
				return
			}
		}
	}))
	defer mockServer.Close()

	// Create a new client with the mock server's URL
	client := NewOpenAIClient("test_api_key", "test_model")
	client.Client.BaseURL = mockServer.URL

	// Test case: Successful streaming prompt
	req := PromptRequest{
		Prompt:      "Hello",
		MaxTokens:   50,
		Temperature: 0.7,
	}
	content, err := client.PromptStream(context.Background(), req)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	if content != "Hello, how can I help you?" {
		t.Errorf("Expected content 'Hello, how can I help you?', but got '%s'", content)
	}
}
