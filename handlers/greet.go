package handlers

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/MatthewJM96/susnames/components"
)

func New(log *slog.Logger) *DefaultHandler {
	return &DefaultHandler{
		Log: log,
	}
}

type DefaultHandler struct {
	Log *slog.Logger
}

func (h *DefaultHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		h.Post(w, r)
		return
	}
	h.Get(w, r)
}

func (h *DefaultHandler) Get(w http.ResponseWriter, r *http.Request) {
	println()
	components.Page(components.HelloPage("John")).Render(r.Context(), os.Stdout)
	println()

	components.Page(components.HelloPage("John")).Render(r.Context(), w)
}

func (h *DefaultHandler) Post(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	name := "John"
	if r.Form.Has("name") {
		name = r.FormValue("name")
	}

	println()
	components.Greeting(name).Render(r.Context(), os.Stdout)
	println()

	components.Greeting(name).Render(r.Context(), w)
}
