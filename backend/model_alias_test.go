package backend

import (
	"context"
	"testing"

	"llm_proxy/models"
)

type aliasFakeBackend struct {
	chatModel     string
	generateModel string
	showModel     string
	models        []models.ModelInfo
}

func (f *aliasFakeBackend) Chat(_ context.Context, req models.ChatRequest) (<-chan models.ChatResponse, *BackendMetadata, error) {
	f.chatModel = req.Model
	out := make(chan models.ChatResponse, 1)
	out <- models.ChatResponse{Model: req.Model}
	close(out)
	return out, &BackendMetadata{}, nil
}

func (f *aliasFakeBackend) Generate(_ context.Context, req models.GenerateRequest) (<-chan models.GenerateResponse, *BackendMetadata, error) {
	f.generateModel = req.Model
	out := make(chan models.GenerateResponse, 1)
	out <- models.GenerateResponse{Model: req.Model}
	close(out)
	return out, &BackendMetadata{}, nil
}

func (f *aliasFakeBackend) ListModels(context.Context) (models.ModelsResponse, error) {
	return models.ModelsResponse{Models: f.models}, nil
}

func (f *aliasFakeBackend) ShowModel(_ context.Context, model string) (models.ShowResponse, error) {
	f.showModel = model
	return models.ShowResponse{}, nil
}

func TestModelAliasPublishesOnlyConfiguredLoadedModel(t *testing.T) {
	inner := &aliasFakeBackend{models: []models.ModelInfo{
		{Name: "other", Model: "other"},
		{Name: "physical-gemma", Model: "physical-gemma", OpenAI: &models.OpenAIModelInfo{ID: "physical-gemma"}},
	}}
	alias := NewModelAliasBackend(inner, "gemma4", "physical-gemma")
	got, err := alias.ListModels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Models) != 1 || got.Models[0].Name != "gemma4" || got.Models[0].Model != "gemma4" {
		t.Fatalf("published models = %+v", got.Models)
	}
	if got.Models[0].OpenAI == nil || got.Models[0].OpenAI.ID != "gemma4" {
		t.Fatalf("OpenAI model ID = %+v", got.Models[0].OpenAI)
	}
	if inner.models[1].Name != "physical-gemma" || inner.models[1].OpenAI.ID != "physical-gemma" {
		t.Fatal("upstream catalogue was mutated")
	}
}

func TestModelAliasTranslatesRequestsAndResponses(t *testing.T) {
	inner := &aliasFakeBackend{}
	alias := NewModelAliasBackend(inner, "gemma4", "physical-gemma")
	chat, metadata, err := alias.Chat(context.Background(), models.ChatRequest{Model: "gemma4"})
	if err != nil || metadata == nil || inner.chatModel != "physical-gemma" {
		t.Fatalf("Chat translation: upstream=%q metadata=%v err=%v", inner.chatModel, metadata, err)
	}
	if got := <-chat; got.Model != "gemma4" {
		t.Fatalf("Chat response model = %q", got.Model)
	}
	generated, metadata, err := alias.Generate(context.Background(), models.GenerateRequest{Model: "gemma4"})
	if err != nil || metadata == nil || inner.generateModel != "physical-gemma" {
		t.Fatalf("Generate translation: upstream=%q metadata=%v err=%v", inner.generateModel, metadata, err)
	}
	if got := <-generated; got.Model != "gemma4" {
		t.Fatalf("Generate response model = %q", got.Model)
	}
	_, err = alias.ShowModel(context.Background(), "gemma4")
	if err != nil || inner.showModel != "physical-gemma" {
		t.Fatalf("Show translation: upstream=%q err=%v", inner.showModel, err)
	}
}

func TestModelAliasRejectsOtherNamesAndAbsentUpstream(t *testing.T) {
	inner := &aliasFakeBackend{models: []models.ModelInfo{{Name: "other", Model: "other"}}}
	alias := NewModelAliasBackend(inner, "gemma4", "physical-gemma")
	if _, err := alias.ListModels(context.Background()); err == nil {
		t.Fatal("missing upstream model was advertised")
	}
	if _, metadata, err := alias.Chat(context.Background(), models.ChatRequest{Model: "physical-gemma"}); err == nil || metadata == nil {
		t.Fatalf("physical name accepted or metadata nil: %v %v", err, metadata)
	}
	if _, metadata, err := alias.Generate(context.Background(), models.GenerateRequest{Model: "other"}); err == nil || metadata == nil {
		t.Fatalf("other name accepted or metadata nil: %v %v", err, metadata)
	}
	if _, err := alias.ShowModel(context.Background(), "physical-gemma"); err == nil {
		t.Fatal("physical name accepted for ShowModel")
	}
}
