package cookies

import (
	"errors"
	"net/http"
	"time"
)

var ErrRefreshCookieMissing = errors.New("refresh cookie is missing")

type Config struct {
	Name     string
	Domain   string
	Path     string
	Secure   bool
	HTTPOnly bool
	SameSite http.SameSite
}

func SetRefreshToken(w http.ResponseWriter, cfg Config, token string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     cfg.Name,
		Value:    token,
		Path:     cfg.Path,
		Domain:   cfg.Domain,
		Expires:  time.Now().Add(ttl),
		MaxAge:   int(ttl.Seconds()),
		Secure:   cfg.Secure,
		HttpOnly: cfg.HTTPOnly,
		SameSite: cfg.SameSite,
	})
}

func ClearRefreshToken(w http.ResponseWriter, cfg Config) {
	http.SetCookie(w, &http.Cookie{
		Name:     cfg.Name,
		Value:    "",
		Path:     cfg.Path,
		Domain:   cfg.Domain,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		Secure:   cfg.Secure,
		HttpOnly: cfg.HTTPOnly,
		SameSite: cfg.SameSite,
	})
}

func ReadRefreshToken(r *http.Request, cfg Config) (string, error) {
	cookie, err := r.Cookie(cfg.Name)
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return "", ErrRefreshCookieMissing
		}
		return "", err
	}
	if cookie.Value == "" {
		return "", ErrRefreshCookieMissing
	}

	return cookie.Value, nil
}
