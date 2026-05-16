package groq

import (
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/httpclient"
)

type GroqProvider struct {
	client *httpclient.Client
}

func NewGroqProvider(
	apiKey string,
) *GroqProvider {
	return &GroqProvider{
		client: httpclient.NewClient(
			apiKey,
			"https://api.groq.com/openai/v1",
			30*time.Second,
		),
	}
}

func (p *GroqProvider) Name() string {
	return "groq"
}

