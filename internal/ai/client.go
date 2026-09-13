package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

type LLMClient struct {
	httpClient     *http.Client
	baseURL        string
	apiKey         string
	model          string
	whisperBaseURL string
	whisperAPIKey  string
	whisperModel   string
}

type LLMResponse struct {
	Content   string
	ToolCalls []ToolCall
}

type ToolCall struct {
	Name      string
	Arguments json.RawMessage
}

func NewLLMClient(baseURL, apiKey, model string) *LLMClient {
	return &LLMClient{
		httpClient:     &http.Client{Timeout: 45 * time.Second},
		baseURL:        strings.TrimSpace(baseURL),
		apiKey:         strings.TrimSpace(apiKey),
		model:          strings.TrimSpace(model),
		whisperBaseURL: strings.TrimSpace(baseURL),
		whisperAPIKey:  strings.TrimSpace(apiKey),
		whisperModel:   "whisper-1",
	}
}

// SetWhisperConfig updates the Whisper transcription configuration.
func (c *LLMClient) SetWhisperConfig(baseURL, apiKey, model string) {
	if strings.TrimSpace(baseURL) != "" {
		c.whisperBaseURL = strings.TrimSpace(baseURL)
	}
	if strings.TrimSpace(apiKey) != "" {
		c.whisperAPIKey = strings.TrimSpace(apiKey)
	}
	if strings.TrimSpace(model) != "" {
		c.whisperModel = strings.TrimSpace(model)
	}
}

// TranscribeAudio sends an audio file to an OpenAI-compatible /audio/transcriptions endpoint.
func (c *LLMClient) TranscribeAudio(ctx context.Context, audioBytes []byte, filename string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("llm client is nil")
	}
	if len(audioBytes) == 0 {
		return "", fmt.Errorf("audio content is empty")
	}
	if filename == "" {
		filename = "voice_note.ogg"
	}

	whisperURL := c.whisperBaseURL
	if whisperURL == "" {
		whisperURL = c.baseURL
	}
	whisperKey := c.whisperAPIKey
	if whisperKey == "" {
		whisperKey = c.apiKey
	}
	whisperModel := c.whisperModel
	if whisperModel == "" {
		whisperModel = "whisper-1"
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", fmt.Errorf("failed to create multipart form file: %w", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(audioBytes)); err != nil {
		return "", fmt.Errorf("failed to write audio bytes to form: %w", err)
	}

	if err := writer.WriteField("model", whisperModel); err != nil {
		return "", fmt.Errorf("failed to write model field: %w", err)
	}
	_ = writer.WriteField("language", "id")

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	endpoint := strings.TrimRight(whisperURL, "/") + "/audio/transcriptions"
	reqCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, body)
	if err != nil {
		return "", fmt.Errorf("failed to create transcription request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+whisperKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("transcription request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read transcription response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("transcription failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBytes)))
	}

	var transcriptionResp struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(respBytes, &transcriptionResp); err != nil {
		return "", fmt.Errorf("failed to parse transcription response: %w", err)
	}

	text := strings.TrimSpace(transcriptionResp.Text)
	if text == "" {
		return "", fmt.Errorf("transcription returned empty text")
	}

	return text, nil
}

// Chat sends one system + one user message to an OpenAI-compatible endpoint.
// It supports regular text responses and function/tool call responses.
func (c *LLMClient) Chat(ctx context.Context, systemPrompt, userMessage string) (*LLMResponse, error) {
	if c == nil {
		return nil, fmt.Errorf("llm client is nil")
	}
	if c.baseURL == "" {
		return nil, fmt.Errorf("baseURL is empty")
	}
	if c.apiKey == "" {
		return nil, fmt.Errorf("apiKey is empty")
	}
	if c.model == "" {
		return nil, fmt.Errorf("model is empty")
	}

	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	payload := openAIChatRequest{
		Model: c.model,
		Messages: []openAIMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMessage},
		},
		Tools:      ToolDefinitions,
		ToolChoice: "auto",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := strings.TrimRight(c.baseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("llm request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read llm response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("llm request returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed openAIChatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("failed to decode llm response json: %w", err)
	}

	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("empty llm response: no choices")
	}

	msg := parsed.Choices[0].Message
	result := &LLMResponse{
		Content: strings.TrimSpace(msg.Content),
	}

	// Prioritize tool calls if present.
	if len(msg.ToolCalls) > 0 {
		result.ToolCalls = make([]ToolCall, 0, len(msg.ToolCalls))
		for _, tc := range msg.ToolCalls {
			if strings.TrimSpace(tc.Function.Name) == "" {
				return nil, fmt.Errorf("invalid tool call: missing function name")
			}

			args := strings.TrimSpace(tc.Function.Arguments)
			if args == "" {
				args = "{}"
			}
			if !json.Valid([]byte(args)) {
				return nil, fmt.Errorf("invalid tool call arguments for %s: malformed json", tc.Function.Name)
			}

			result.ToolCalls = append(result.ToolCalls, ToolCall{
				Name:      tc.Function.Name,
				Arguments: json.RawMessage(args),
			})
		}
		return result, nil
	}

	if result.Content == "" {
		return nil, fmt.Errorf("empty llm response: no content and no tool calls")
	}

	return result, nil
}

type openAIChatRequest struct {
	Model      string                   `json:"model"`
	Messages   []openAIMessage          `json:"messages"`
	Tools      []map[string]interface{} `json:"tools,omitempty"`
	ToolChoice string                   `json:"tool_choice,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatResponse struct {
	Choices []openAIChoice `json:"choices"`
}

type openAIChoice struct {
	Message openAIResponseMessage `json:"message"`
}

type openAIResponseMessage struct {
	Content   string           `json:"content"`
	ToolCalls []openAIToolCall `json:"tool_calls"`
}

type openAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function openAIToolFunction `json:"function"`
}

type openAIToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}
