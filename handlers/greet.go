package handlers

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/MatthewJM96/susnames/components"
	"github.com/spf13/viper"
)

func New(config *viper.Viper, log *slog.Logger) *DefaultHandler {
	return &DefaultHandler{
		Config: config,
		Log:    log,
	}
}

type DefaultHandler struct {
	Config *viper.Viper
	Log    *slog.Logger
}

func (h *DefaultHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		h.Post(w, r)
		return
	}
	h.Get(w, r)
}

func (h *DefaultHandler) Get(w http.ResponseWriter, r *http.Request) {
	if h.Config.GetBool("debug") {
		println()
		components.Page(components.HelloPage("John")).Render(r.Context(), os.Stdout)
		println()
	}

	components.Page(components.HelloPage("John")).Render(r.Context(), w)
}

func (h *DefaultHandler) Post(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	name := "John"
	if r.Form.Has("name") {
		name = r.FormValue("name")
	}

	if h.Config.GetBool("debug") {
		println()
		components.Greeting(name).Render(r.Context(), os.Stdout)
		println()
	}

	components.Greeting(name).Render(r.Context(), w)
}
