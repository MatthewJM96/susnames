package session

import (
	"net/http"
	"time"

	"github.com/segmentio/ksuid"
	"github.com/spf13/viper"
)

type SessionMiddlewareOpts func(*SessionMiddleware)

var sessionID string = ""

func NewSessionMiddleware(next http.Handler, config *viper.Viper) http.Handler {
	return SessionMiddleware{
		Next:     next,
		Secure:   config.GetBool("secure"),
		HTTPOnly: config.GetBool("http_only"),
	}
}

func WithSecure(secure bool) SessionMiddlewareOpts {
	return func(m *SessionMiddleware) {
		m.Secure = secure
	}
}

func WithHTTPOnly(httpOnly bool) SessionMiddlewareOpts {
	return func(m *SessionMiddleware) {
		m.HTTPOnly = httpOnly
	}
}

type SessionMiddleware struct {
	Next     http.Handler
	Secure   bool
	HTTPOnly bool
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
			Secure:   mw.Secure,
			HttpOnly: mw.HTTPOnly,
			Expires:  time.Now().Add(30 * 24 * time.Hour),
			Path:     "/",
		},
	)
}

func (mw SessionMiddleware) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
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

	mw.Next.ServeHTTP(writer, request)
}
