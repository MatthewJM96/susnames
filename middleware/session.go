package middleware

import (
	"net/http"

	"github.com/segmentio/ksuid"
	"github.com/spf13/viper"
)

type SessionMiddlewareOpts func(*SessionMiddleware)

func NewSessionMiddleware(next http.Handler, config *viper.Viper) http.Handler {
	mw := SessionMiddleware{
		Next:     next,
		Secure:   config.GetBool("secure"),
		HTTPOnly: config.GetBool("http_only"),
	}
	return mw
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

func GetID(r *http.Request) string {
	cookie, err := r.Cookie("sessionID")
	if err != nil {
		return ""
	}
	return cookie.Value
}

func (mw SessionMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := GetID(r)
	if id == "" {
		id = ksuid.New().String()
		http.SetCookie(w, &http.Cookie{Name: "sessionID", Value: id, Secure: mw.Secure, HttpOnly: mw.HTTPOnly})
	}

	mw.Next.ServeHTTP(w, r)
}
