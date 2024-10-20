package handlers

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/MatthewJM96/susnames/components"
	"github.com/spf13/viper"
)

func NewGridHandler(config *viper.Viper, log *slog.Logger) *GridHandler {
	return &GridHandler{
		Config: config,
		Log:    log,
	}
}

type GridHandler struct {
	Config *viper.Viper
	Log    *slog.Logger
}

func (h *GridHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		h.Post(w, r)
		return
	}
	h.Get(w, r)
}

func (h *GridHandler) Get(w http.ResponseWriter, r *http.Request) {
	if h.Config.GetBool("debug") {
		println()

		components.Page(components.HelloPage("John")).Render(r.Context(), os.Stdout)

		println()
	}

	components.Page(
		components.Room(
			"test-room",
			components.Grid([25]string{"relinquish", "genuine", "formula", "gain", "established", "development", "long", "personality", "package", "reveal", "premium", "carve", "authority", "blast", "compromise", "acid", "video", "live", "eject", "redundancy", "announcement", "tear", "depressed", "cunning", "child"}),
		),
	).Render(r.Context(), w)
}

func (h *GridHandler) Post(w http.ResponseWriter, r *http.Request) {

}
