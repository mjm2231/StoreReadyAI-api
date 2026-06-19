package ai

import "context"

const (
	ProviderGoogleGemini = "google_gemini"
)

// Config AI 客户端配置。
//
// MVP 默认使用 Google Gemini，后续可通过 Provider 切换到 OpenAI、Claude 或自建模型。
type Config struct {
	Provider       string
	Model          string
	APIKey         string
	Endpoint       string
	TimeoutSeconds int
}

// StoreInfoInput 生成上架资料的输入上下文。
type StoreInfoInput struct {
	ProjectID   uint64
	Name        string
	Description string
	Platform    string
	Status      string

	ExistingAppName          string
	ExistingSubtitle         string
	ExistingKeywords         string
	ExistingShortDescription string
	ExistingFullDescription  string
	ExistingCategory         string
	ExistingContentRating    string
}

// GeneratedStoreInfo AI 生成出的上架资料文本。
//
// App 名称只作为 AI 上下文输入，不作为 AI 生成字段返回，避免覆盖用户手动填写的应用名称。
// AI 只生成副标题、简短描述、完整描述和关键词 4 个字段，前端收到后回填表单，由用户确认后再保存。
type GeneratedStoreInfo struct {
	Subtitle         string `json:"subtitle"`
	ShortDescription string `json:"short_description"`
	FullDescription  string `json:"full_description"`
	Keywords         string `json:"keywords"`
}

// StoreInfoGenerator 上架资料生成器抽象。
//
// 任何 AI provider 只要实现这个接口，就可以接入项目 Service。
type StoreInfoGenerator interface {
	GenerateStoreInfo(ctx context.Context, input StoreInfoInput) (*GeneratedStoreInfo, error)
}

// NewStoreInfoGenerator 根据配置创建上架资料生成器。
func NewStoreInfoGenerator(cfg Config) (StoreInfoGenerator, error) {
	provider := cfg.Provider
	if provider == "" {
		provider = ProviderGoogleGemini
	}

	switch provider {
	case ProviderGoogleGemini:
		return NewGeminiClient(cfg)
	default:
		return nil, ErrUnsupportedProvider
	}
}
