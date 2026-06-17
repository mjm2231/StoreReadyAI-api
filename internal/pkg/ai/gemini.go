package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

var (
	ErrUnsupportedProvider = errors.New("unsupported ai provider")
	ErrAIAPIKeyRequired    = errors.New("ai api key required")
	ErrAIEmptyResponse     = errors.New("ai empty response")
)

const (
	defaultGeminiModel    = "gemini-2.5-flash"
	defaultGeminiEndpoint = "https://generativelanguage.googleapis.com/v1beta/models"
	defaultTimeoutSeconds = 30
)

type GeminiClient struct {
	model    string
	apiKey   string
	endpoint string
	client   *http.Client
}

func NewGeminiClient(cfg Config) (*GeminiClient, error) {
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		return nil, ErrAIAPIKeyRequired
	}

	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = defaultGeminiModel
	}

	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		endpoint = defaultGeminiEndpoint
	}
	endpoint = strings.TrimRight(endpoint, "/")

	timeoutSeconds := cfg.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = defaultTimeoutSeconds
	}

	return &GeminiClient{
		model:    model,
		apiKey:   apiKey,
		endpoint: endpoint,
		client: &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		},
	}, nil
}

func (c *GeminiClient) GenerateStoreInfo(ctx context.Context, input StoreInfoInput) (*GeneratedStoreInfo, error) {
	if c == nil || c.client == nil {
		return nil, errors.New("gemini client not configured")
	}

	prompt := buildStoreInfoPrompt(input)
	payload := geminiGenerateRequest{
		Contents: []geminiContent{
			{
				Role: "user",
				Parts: []geminiPart{
					{Text: prompt},
				},
			},
		},
		GenerationConfig: geminiGenerationConfig{
			Temperature:      0.7,
			MaxOutputTokens:  3000,
			ResponseMIMEType: "application/json",
		},
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/%s:generateContent?key=%s", c.endpoint, c.model, c.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("gemini api failed: status=%d body=%s", resp.StatusCode, string(respBytes))
	}

	var geminiResp geminiGenerateResponse
	if err := json.Unmarshal(respBytes, &geminiResp); err != nil {
		return nil, err
	}

	text := extractGeminiText(geminiResp)
	if strings.TrimSpace(text) == "" {
		return nil, ErrAIEmptyResponse
	}

	generated, err := parseGeneratedStoreInfo(text)
	if err != nil {
		return nil, err
	}
	normalizeGeneratedStoreInfo(generated)
	return generated, nil
}

type geminiGenerateRequest struct {
	Contents         []geminiContent        `json:"contents"`
	GenerationConfig geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	Temperature      float64 `json:"temperature,omitempty"`
	MaxOutputTokens  int     `json:"maxOutputTokens,omitempty"`
	ResponseMIMEType string  `json:"responseMimeType,omitempty"`
}

type geminiGenerateResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
}

func buildStoreInfoPrompt(input StoreInfoInput) string {
	return fmt.Sprintf(`你是专业的 App Store / Google Play 上架资料撰写助手。
请根据下面项目信息，生成一份适合应用商店上架的中文资料。

只允许返回 JSON，不要返回 Markdown，不要添加解释。
JSON 字段必须严格为：
{
  "app_name": "",
  "subtitle": "",
  "short_description": "",
  "full_description": "",
  "keywords": ""
}

生成要求：
1. app_name：简洁清晰，优先使用已有 App 名称或项目名称，最多 30 个中文字符。
2. subtitle：一句话说明产品价值，最多 30 个中文字符。
3. short_description：简短描述，适合商店摘要，最多 80 个中文字符。
4. full_description：完整介绍，突出目标用户、核心功能和使用场景，最多 500 个中文字符。
5. keywords：关键词用英文逗号分隔，最多 10 个关键词。
6. 不要夸大，不要承诺无法验证的效果。
7. 不要输出除 JSON 外的任何内容。
8. 不要使用 Markdown，不要使用代码块标记。
9. 必须返回完整可解析 JSON，不能截断。
项目信息：
- project_id: %d
- project_name: %s
- platform: %s
- project_status: %s

已有上架资料：
- app_name: %s
- subtitle: %s
- keywords: %s
- short_description: %s
- full_description: %s
- category: %s
- content_rating: %s
`,
		input.ProjectID,
		input.Name,
		input.Platform,
		input.Status,
		input.ExistingAppName,
		input.ExistingSubtitle,
		input.ExistingKeywords,
		input.ExistingShortDescription,
		input.ExistingFullDescription,
		input.ExistingCategory,
		input.ExistingContentRating,
	)
}

func extractGeminiText(resp geminiGenerateResponse) string {
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return ""
	}
	return resp.Candidates[0].Content.Parts[0].Text
}

func parseGeneratedStoreInfo(text string) (*GeneratedStoreInfo, error) {
	cleaned := cleanJSONText(text)
	if strings.TrimSpace(cleaned) == "" {
		log.Printf("[gemini_parse_store_info_empty] raw=%q cleaned=%q", text, cleaned)
		return nil, ErrAIEmptyResponse
	}

	var generated GeneratedStoreInfo
	if err := json.Unmarshal([]byte(cleaned), &generated); err != nil {
		log.Printf("[gemini_parse_store_info_failed] raw=%q cleaned=%q err=%v", text, cleaned, err)
		return nil, err
	}
	return &generated, nil
}

func cleanJSONText(text string) string {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start >= 0 && end >= start {
		return text[start : end+1]
	}
	return text
}

func normalizeGeneratedStoreInfo(info *GeneratedStoreInfo) {
	if info == nil {
		return
	}
	info.AppName = strings.TrimSpace(info.AppName)
	info.Subtitle = strings.TrimSpace(info.Subtitle)
	info.ShortDescription = strings.TrimSpace(info.ShortDescription)
	info.FullDescription = strings.TrimSpace(info.FullDescription)
	info.Keywords = strings.TrimSpace(info.Keywords)
}
