package ai_agent

import (
	"context"
	"fmt"
	"os"
	"lark_ai/config"
	"sync"
)

// ModelCreator defines model creation function type (requires context)
type ModelCreator func(ctx context.Context, config map[string]any) (AIModel, error)

// AIModelFactory AI model factory
type AIModelFactory struct {
	creators map[string]ModelCreator
}

var (
	globalFactory *AIModelFactory
	factoryOnce   sync.Once
)

// GetGlobalFactory gets global singleton
func GetGlobalFactory() *AIModelFactory {
	factoryOnce.Do(func() {
		globalFactory = &AIModelFactory{
			creators: make(map[string]ModelCreator),
		}
		globalFactory.registerCreators()
	})
	return globalFactory
}

// Register models
func resolveOllamaConfig(cfg map[string]any) (string, string) {
	conf := config.GetConfig().OllamaConfig

	baseURL, _ := cfg["baseURL"].(string)
	if baseURL == "" {
		baseURL = os.Getenv("OLLAMA_BASE_URL")
	}
	if baseURL == "" {
		baseURL = conf.BaseURL
	}
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	modelName, _ := cfg["modelName"].(string)
	if modelName == "" {
		modelName = os.Getenv("OLLAMA_MODEL_NAME")
	}
	if modelName == "" {
		modelName = conf.ModelName
	}
	if modelName == "" {
		modelName = "qwen3"
	}

	return baseURL, modelName
}

func (f *AIModelFactory) registerCreators() {
	f.creators["ollama"] = func(ctx context.Context, cfg map[string]any) (AIModel, error) {
		baseURL, modelName := resolveOllamaConfig(cfg)
		return NewOllamaModel(ctx, baseURL, modelName)
	}

	f.creators["ollama-rag"] = func(ctx context.Context, cfg map[string]any) (AIModel, error) {
		username, ok := cfg["username"].(string)
		if !ok {
			return nil, fmt.Errorf("ollama RAG model requires username")
		}
		baseURL, modelName := resolveOllamaConfig(cfg)
		return NewOllamaRAGModel(ctx, baseURL, modelName, username)
	}
}

// CreateAIModel creates AI model by type
func (f *AIModelFactory) CreateAIModel(ctx context.Context, modelType string, config map[string]any) (AIModel, error) {
	creator, ok := f.creators[modelType]
	if !ok {
		return nil, fmt.Errorf("unsupported model type: %s", modelType)
	}
	return creator(ctx, config)
}

// CreateAIAgent creates AIAgent in one click
func (f *AIModelFactory) CreateAIAgent(ctx context.Context, modelType string, SessionID string, config map[string]any) (*AIAgent, error) {
	model, err := f.CreateAIModel(ctx, modelType, config)
	if err != nil {
		return nil, err
	}
	return NewAIAgent(model, SessionID), nil
}

// RegisterModel supports extensible registration
func (f *AIModelFactory) RegisterModel(modelType string, creator ModelCreator) {
	f.creators[modelType] = creator
}
