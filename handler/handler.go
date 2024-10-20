package handler

import (
	"log/slog"

	"github.com/spf13/viper"
)

func NewHandler(config *viper.Viper, log *slog.Logger) *Handler {
	return &Handler{
		Config: config,
		Log:    log,
	}
}

type Handler struct {
	Config *viper.Viper
	Log    *slog.Logger
}
