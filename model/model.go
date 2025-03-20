package model

import (
	"context"
	"fmt"

	"github.com/bootun/commander/config"
	"github.com/cloudwego/eino-ext/components/model/openai"
	libOpenai "github.com/cloudwego/eino-ext/libs/acl/openai"
)

type Team struct {
	Reasoning  *openai.ChatModel
	Structured *openai.ChatModel
	Security   *openai.ChatModel
	Actor      *openai.ChatModel
}

// NewTeam 初始化模型团队
func NewTeam(ctx context.Context, cfg *config.Config) (*Team, error) {
	reasoningModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: cfg.Models.ReasoningModel.BaseURL,
		APIKey:  cfg.Models.ReasoningModel.Token,
		Model:   cfg.Models.ReasoningModel.ModelID,
	})
	if err != nil {
		return nil, fmt.Errorf("推理模型初始化失败: %v", err)
	}
	structuredModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: cfg.Models.StructuredModel.BaseURL,
		APIKey:  cfg.Models.StructuredModel.Token,
		Model:   cfg.Models.StructuredModel.ModelID,
		ResponseFormat: &libOpenai.ChatCompletionResponseFormat{
			Type: libOpenai.ChatCompletionResponseFormatTypeJSONObject,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("结构化模型初始化失败: %v", err)
	}
	securityModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: cfg.Models.SecurityModel.BaseURL,
		APIKey:  cfg.Models.SecurityModel.Token,
		Model:   cfg.Models.SecurityModel.ModelID,
		ResponseFormat: &libOpenai.ChatCompletionResponseFormat{
			Type: libOpenai.ChatCompletionResponseFormatTypeJSONObject,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("安全模型初始化失败: %v", err)
	}
	actorModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: cfg.Models.ActorModel.BaseURL,
		APIKey:  cfg.Models.ActorModel.Token,
		Model:   cfg.Models.ActorModel.ModelID,
	})
	if err != nil {
		return nil, fmt.Errorf("回答模型初始化失败: %v", err)
	}
	return &Team{
		Reasoning:  reasoningModel,
		Structured: structuredModel,
		Security:   securityModel,
		Actor:      actorModel,
	}, nil
}

func (t *Team) ReasoningModel() *openai.ChatModel {
	return t.Reasoning
}

func (t *Team) StructuredModel() *openai.ChatModel {
	return t.Structured
}

func (t *Team) SecurityModel() *openai.ChatModel {
	return t.Security
}

func (t *Team) ActorModel() *openai.ChatModel {
	return t.Actor
}
