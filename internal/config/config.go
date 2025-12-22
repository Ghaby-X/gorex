// Package config parses the yaml configuration file
package config

import (
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/go-yaml/yaml"
)

// Config defines trading configuration
type Config struct {
	Provider ProviderConfig `yaml:"provider" validate:"required"`
	Executor ExecutorConfig `yaml:"executor" validate:"required"`
	Assets   []AssetConfig  `yaml:"assets" validate:"required,min=1,dive"`
}

type ProviderConfig struct {
	Name      string `yaml:"name" validate:"required,oneof=metatrader5"`
	Type      string `yaml:"type" validate:"required,oneof=candle"`
	Timeframe string `yaml:"timeframe" validate:"required_if=Type candle"`
}

type ExecutorConfig struct {
	Name string `yaml:"name" validate:"required"`
}

type AssetConfig struct {
	Symbol   string         `yaml:"symbol" validate:"required"`
	Strategy StrategyConfig `yaml:"strategy" validate:"required"`
	Risk     RiskConfig     `yaml:"risk" validate:"required"`
}

type StrategyConfig struct {
	Name   string         `yaml:"name"`
	Params map[string]any `yaml:"params"`
}

type RiskConfig struct {
	TakeProfitPips int     `yaml:"take_profit_pips" validate:"required,gt=0"`
	StopLossPips   int     `yaml:"stop_loss_pips"   validate:"required,gt=0"`
	LotSize        int     `yaml:"lot_size"        validate:"required,min=1000"`
	Leverage       float64 `yaml:"leverage"        validate:"required,gt=0"`
}

func LoadConfig(filename string) (*Config, error) {
	yamlfile, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = yaml.Unmarshal(yamlfile, &cfg)
	if err != nil {
		return nil, err
	}

	// validate struct
	validate := validator.New()
	err = validate.Struct(&cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
