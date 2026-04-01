package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// GeminiProvider uses the Google Gemini API for completions.
type GeminiProvider struct {
	apiKey string
}

// NewGeminiProvider creates a Gemini provider using GEMINI_API_KEY.
func NewGeminiProvider() (*GeminiProvider, error) {
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY not set")
	}
	return &GeminiProvider{apiKey: key}, nil
}

func (p *GeminiProvider) Name() string { return "gemini" }

func (p *GeminiProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	start := time.Now()

	model := req.Model
	if model == "" {
		model = "gemini-2.5-flash"
	}

	contents := []map[string]any{}
	if req.SystemPrompt != "" {
		contents = append(contents, map[string]any{
			"role":  "user",
			"parts": []map[string]string{{"text": fmt.Sprintf("[System instructions: %s]\n\n%s", req.SystemPrompt, req.UserPrompt)}},
		})
	} else {
		contents = append(contents, map[string]any{
			"role":  "user",
			"parts": []map[string]string{{"text": req.UserPrompt}},
		})
	}

	body := map[string]any{"contents": contents}
	jsonBody, _ := json.Marshal(body)

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", model)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", p.apiKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gemini request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("gemini HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
		} `json:"usageMetadata"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("gemini parse: %w", err)
	}

	content := ""
	if len(result.Candidates) > 0 && len(result.Candidates[0].Content.Parts) > 0 {
		content = result.Candidates[0].Content.Parts[0].Text
	}

	return &CompletionResponse{
		Content: content,
		Latency: time.Since(start),
		Tokens: TokenUsage{
			Input:  result.UsageMetadata.PromptTokenCount,
			Output: result.UsageMetadata.CandidatesTokenCount,
		},
	}, nil
}
