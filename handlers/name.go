package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/MatthewJM96/susnames/components"
	"github.com/goombaio/namegenerator"
	"github.com/spf13/viper"
)

func NewNameHandler(config *viper.Viper, log *slog.Logger) *NameHandler {
	return &NameHandler{
		Config: config,
		Log:    log,
	}
}

type NameHandler struct {
	Config *viper.Viper
	Log    *slog.Logger
}

func (h *NameHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.Get(w, r)
	} else if r.Method == http.MethodPost {
		h.Post(w, r)
	} else {
		http.Error(w, "only supports Get/POST", 405)
	}
}

func genName() string {
	seed := time.Now().UTC().UnixNano()
	nameGenerator := namegenerator.NewNameGenerator(seed)

	return nameGenerator.Generate()
}

func getName(r *http.Request) string {
	cookie, err := r.Cookie("name")
	if err != nil {
		return genName()
	}
	return cookie.Value
}

func (h *NameHandler) Get(w http.ResponseWriter, r *http.Request) {
	name := getName(r)

	components.Greeting(name).Render(r.Context(), w)
}

func (h *NameHandler) Post(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	name := genName()
	if r.Form.Has("name") {
		name = r.FormValue("name")

		h.Log.Info(fmt.Sprintf("set player name: %s", name))
	} else {
		h.Log.Info(fmt.Sprintf("set player name to default: %s", name))
	}

	http.SetCookie(w, &http.Cookie{Name: "name", Value: name, Secure: h.Config.GetBool("secure"), HttpOnly: h.Config.GetBool("http_only")})

	components.Greeting(name).Render(r.Context(), w)
}
