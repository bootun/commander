// 这个文件中的内容是AI生成的
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// 配置
type Config struct {
	Models    Models `yaml:"models"`
	MaxRounds int    `yaml:"max_rounds"`
}

// 模型列表
type Models struct {
	ReasoningModel  Model `yaml:"reasoning_model"`  // 推理模型
	StructuredModel Model `yaml:"structured_model"` // 结构化模型
	SecurityModel   Model `yaml:"security_model"`   // 安全模型
	ActorModel      Model `yaml:"actor_model"`      // 回答模型
}

// 单个模型的配置
type Model struct {
	ModelID string `yaml:"model_id"`
	BaseURL string `yaml:"base_url"`
	Token   string `yaml:"token"`
}

// LoadConfig loads the configuration from the specified file path.
// If filePath is empty, it will look for config.yml in the current directory.
func LoadConfig(filePath string) (*Config, error) {
	if filePath == "" {
		filePath = "config.yml"
	}

	// If not an absolute path, resolve to absolute
	if !filepath.IsAbs(filePath) {
		// Get the current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current working directory: %w", err)
		}
		filePath = filepath.Join(cwd, filePath)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate essential configuration
	if err := validateModelConfig(cfg.Models.ReasoningModel, "reasoning"); err != nil {
		return nil, err
	}
	if err := validateModelConfig(cfg.Models.StructuredModel, "structured"); err != nil {
		return nil, err
	}
	if err := validateModelConfig(cfg.Models.SecurityModel, "security"); err != nil {
		return nil, err
	}
	if err := validateModelConfig(cfg.Models.ActorModel, "actor"); err != nil {
		return nil, err
	}
	if cfg.MaxRounds <= 0 {
		cfg.MaxRounds = 5
	}

	return &cfg, nil
}

// validateModelConfig validates a model configuration and returns an appropriate error if incomplete
func validateModelConfig(model Model, modelType string) error {
	missing := []string{}

	if model.ModelID == "" {
		missing = append(missing, "model_id")
	}
	if model.BaseURL == "" {
		missing = append(missing, "base_url")
	}
	if model.Token == "" {
		missing = append(missing, "token")
	}

	if len(missing) > 0 {
		return fmt.Errorf("%s model configuration is incomplete: missing %v", modelType, missing)
	}

	return nil
}
