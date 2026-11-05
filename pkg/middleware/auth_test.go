package middleware_test

import (
	"go-adv-demo/configs"
	"go-adv-demo/pkg/jwt"
	"go-adv-demo/pkg/middleware"
	"net/http"
	"net/http/httptest"
	"testing"
)

const testSecret = "middleware-test-secret"

// protectedHandler оборачивает обработчик в IsAuth и считает, сколько раз
// до него дошёл запрос и какой email лежит в контексте.
type protectedHandler struct {
	handler http.Handler
	calls   int
	email   string
}

func newProtectedHandler(secret string) *protectedHandler {
	p := &protectedHandler{}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p.calls++
		p.email, _ = r.Context().Value(middleware.ContextEmailKey).(string)
		w.WriteHeader(http.StatusOK)
	})
	p.handler = middleware.IsAuth(next, &configs.Config{Auth: configs.AuthConfig{Secret: secret}})
	return p
}

func (p *protectedHandler) serve(authHeader string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	rec := httptest.NewRecorder()
	p.handler.ServeHTTP(rec, req)
	return rec
}

func TestIsAuthRejectsRequestsWithoutValidBearer(t *testing.T) {
	cases := map[string]string{
		"no header":     "",
		"basic scheme":  "Basic dXNlcjpwYXNz",
		"bearer no sp":  "Bearer",
		"garbage token": "Bearer not.a.jwt",
	}
	for name, header := range cases {
		t.Run(name, func(t *testing.T) {
			p := newProtectedHandler(testSecret)
			rec := p.serve(header)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d", rec.Code)
			}
			if p.calls != 0 {
				t.Fatal("next handler must not be called for unauthorized request")
			}
		})
	}
}

func TestIsAuthRejectsTokenSignedWithOtherSecret(t *testing.T) {
	token, err := jwt.NewJWT("some-other-secret").Create(jwt.JWTData{Email: "user@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	p := newProtectedHandler(testSecret)
	rec := p.serve("Bearer " + token)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if p.calls != 0 {
		t.Fatal("next handler must not be called for forged token")
	}
}

func TestIsAuthPassesEmailFromValidToken(t *testing.T) {
	token, err := jwt.NewJWT(testSecret).Create(jwt.JWTData{Email: "user@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	p := newProtectedHandler(testSecret)
	rec := p.serve("Bearer " + token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if p.calls != 1 {
		t.Fatalf("expected next handler to be called once, got %d", p.calls)
	}
	if p.email != "user@example.com" {
		t.Fatalf("expected email in context, got %q", p.email)
	}
}
