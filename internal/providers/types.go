package providers

type Message struct {
	Role    string
	Content string
}

type GenerateRequest struct {
	Model    string
	Messages []Message

	Temperature float64
	MaxTokens   int

	Metadata map[string]any
}

type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type GenerateResponse struct {
	ID           string
	Model        string
	Content      string
	FinishReason string
	Usage        TokenUsage
	Raw          map[string]any
}

type StreamChunk struct {
	Content string
	Done    bool
}
