package session

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/segmentio/ksuid"
	"github.com/spf13/viper"
)

type SessionMiddlewareOpts func(*SessionMiddleware)

var sessionID string = ""

func NewSessionMiddleware(next http.Handler, config *viper.Viper, log *slog.Logger) http.Handler {
	return SessionMiddleware{
		Config: config,
		Log:    log,
		Next:   next,
	}
}

type SessionMiddleware struct {
	Config *viper.Viper
	Log    *slog.Logger

	Next http.Handler
}

func SessionID() string {
	return sessionID
}

func (mw SessionMiddleware) setSessionID(writer http.ResponseWriter) {
	http.SetCookie(
		writer,
		&http.Cookie{
			Name:     "SN-SessionID",
			Value:    sessionID,
			Secure:   mw.Config.GetBool("secure"),
			HttpOnly: mw.Config.GetBool("http_only"),
			Expires:  time.Now().Add(30 * 24 * time.Hour),
			Path:     "/",
		},
	)
}

func (mw SessionMiddleware) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if mw.Config.GetBool("debug") {
		// In debug mode, we don't persist session IDs so that multiple tabs can be used
		// in the same browser.
		sessionID = ksuid.New().String()
	} else {
		cookie, err := request.Cookie("SN-SessionID")
		if err == nil {
			sessionID = cookie.Value

			if cookie.Expires.Compare(time.Now().Add(5*24*time.Hour)) == -1 {
				mw.setSessionID(writer)
			}
		} else {
			sessionID = ksuid.New().String()
			mw.setSessionID(writer)
		}
	}

	mw.Next.ServeHTTP(writer, request)
}
