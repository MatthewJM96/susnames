package handler

import (
	"net/http"

	"github.com/MatthewJM96/susnames/components"
)

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	components.Page(components.Home()).Render(r.Context(), w)
}
