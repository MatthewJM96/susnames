package main

import (
	"fmt"

	"github.com/spf13/viper"
)

func loadConfig() *viper.Viper {
	config := viper.New()

	config.SetConfigFile(".env")
	config.SetConfigType("env")

	config.AddConfigPath(".")

	config.SetDefault("debug", false)
	config.SetDefault("secure", true)
	config.SetDefault("http_only", true)

	err := config.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	return config
}
