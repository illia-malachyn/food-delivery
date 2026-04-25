package cookies

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testCookieConfig() Config {
	return Config{
		Name:     "refresh_token",
		Path:     "/",
		Secure:   true,
		HTTPOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

func TestSetAndReadRefreshToken(t *testing.T) {
	cfg := testCookieConfig()
	recorder := httptest.NewRecorder()

	SetRefreshToken(recorder, cfg, "token-123", time.Hour)

	res := recorder.Result()
	cookies := res.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	if cookies[0].Name != cfg.Name || cookies[0].Value != "token-123" {
		t.Fatalf("unexpected cookie written: %+v", cookies[0])
	}
	if !cookies[0].HttpOnly || !cookies[0].Secure {
		t.Fatal("refresh cookie should be HttpOnly and Secure")
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.AddCookie(cookies[0])
	value, err := ReadRefreshToken(req, cfg)
	if err != nil {
		t.Fatalf("ReadRefreshToken() error = %v", err)
	}
	if value != "token-123" {
		t.Fatalf("unexpected token read: %s", value)
	}
}

func TestClearRefreshToken(t *testing.T) {
	cfg := testCookieConfig()
	recorder := httptest.NewRecorder()

	ClearRefreshToken(recorder, cfg)

	setCookie := recorder.Header().Get("Set-Cookie")
	if setCookie == "" {
		t.Fatal("expected Set-Cookie header")
	}
	if !strings.Contains(setCookie, cfg.Name+"=") {
		t.Fatalf("expected cookie name in Set-Cookie header: %s", setCookie)
	}
	if !strings.Contains(setCookie, "Max-Age=0") {
		t.Fatalf("expected Max-Age=0 in Set-Cookie header: %s", setCookie)
	}
}
