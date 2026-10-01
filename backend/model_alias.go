package backend

import (
	"context"
	"fmt"

	"llm_proxy/models"
)

// ModelAliasBackend exposes one stable public name for one already loaded model.
// It does not select, load, or route among models.
type ModelAliasBackend struct {
	inner    Backend
	public   string
	upstream string
}

func NewModelAliasBackend(inner Backend, public, upstream string) *ModelAliasBackend {
	return &ModelAliasBackend{inner: inner, public: public, upstream: upstream}
}

func (a *ModelAliasBackend) requirePublic(model string) error {
	if model != a.public {
		return fmt.Errorf("model %q is unavailable; use %q", model, a.public)
	}
	return nil
}

func (a *ModelAliasBackend) Generate(ctx context.Context, req models.GenerateRequest) (<-chan models.GenerateResponse, *BackendMetadata, error) {
	if err := a.requirePublic(req.Model); err != nil {
		return nil, &BackendMetadata{}, err
	}
	req.Model = a.upstream
	responses, metadata, err := a.inner.Generate(ctx, req)
	if err != nil {
		if metadata == nil {
			metadata = &BackendMetadata{}
		}
		return nil, metadata, err
	}
	out := make(chan models.GenerateResponse)
	go func() {
		defer close(out)
		for response := range responses {
			response.Model = a.public
			select {
			case out <- response:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, metadata, nil
}

func (a *ModelAliasBackend) Chat(ctx context.Context, req models.ChatRequest) (<-chan models.ChatResponse, *BackendMetadata, error) {
	if err := a.requirePublic(req.Model); err != nil {
		return nil, &BackendMetadata{}, err
	}
	req.Model = a.upstream
	responses, metadata, err := a.inner.Chat(ctx, req)
	if err != nil {
		if metadata == nil {
			metadata = &BackendMetadata{}
		}
		return nil, metadata, err
	}
	out := make(chan models.ChatResponse)
	go func() {
		defer close(out)
		for response := range responses {
			response.Model = a.public
			select {
			case out <- response:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, metadata, nil
}

func (a *ModelAliasBackend) ListModels(ctx context.Context) (models.ModelsResponse, error) {
	response, err := a.inner.ListModels(ctx)
	if err != nil {
		return models.ModelsResponse{}, err
	}
	for _, model := range response.Models {
		if model.Name != a.upstream && model.Model != a.upstream {
			continue
		}
		model.Name = a.public
		model.Model = a.public
		if model.OpenAI != nil {
			copy := *model.OpenAI
			copy.ID = a.public
			model.OpenAI = &copy
		}
		return models.ModelsResponse{Models: []models.ModelInfo{model}}, nil
	}
	return models.ModelsResponse{}, fmt.Errorf("configured upstream model %q is not loaded", a.upstream)
}

func (a *ModelAliasBackend) ShowModel(ctx context.Context, model string) (models.ShowResponse, error) {
	if err := a.requirePublic(model); err != nil {
		return models.ShowResponse{}, err
	}
	return a.inner.ShowModel(ctx, a.upstream)
}
