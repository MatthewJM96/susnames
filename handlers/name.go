package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

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
	if r.Method == http.MethodPost {
		h.Post(w, r)
		return
	}
	http.Error(w, "only supports POST", 405)
}

func (h *NameHandler) Post(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	seed := time.Now().UTC().UnixNano()
	nameGenerator := namegenerator.NewNameGenerator(seed)

	name := nameGenerator.Generate()
	if r.Form.Has("name") {
		name = r.FormValue("name")

		h.Log.Info(fmt.Sprintf("set player name: %s", name))
	} else {
		h.Log.Info(fmt.Sprintf("set player name to default: %s", name))
	}

	http.SetCookie(w, &http.Cookie{Name: "name", Value: name, Secure: h.Config.GetBool("secure"), HttpOnly: h.Config.GetBool("http_only")})
}
