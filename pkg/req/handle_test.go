package req_test

import (
	"go-adv-demo/pkg/req"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

type payload struct {
	Email string `json:"email" validate:"required,email"`
	Name  string `json:"name" validate:"required"`
}

func handle(body string) (*payload, *httptest.ResponseRecorder, error) {
	rec := httptest.NewRecorder()
	var w http.ResponseWriter = rec
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	p, err := req.HandleBody[payload](&w, r)
	return p, rec, err
}

func TestHandleBodyDecodesValidPayload(t *testing.T) {
	p, rec, err := handle(`{"email":"regai@ya.ru","name":"BlackMamba"}`)
	if err != nil {
		t.Fatal(err)
	}
	if p.Email != "regai@ya.ru" || p.Name != "BlackMamba" {
		t.Fatalf("unexpected payload: %+v", p)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("handler must not write a response on success, wrote %q", rec.Body.String())
	}
}

func TestHandleBodyRejectsMalformedJSON(t *testing.T) {
	_, rec, err := handle(`{"email":`)
	if err == nil {
		t.Fatal("expected decode error")
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleBodyRejectsFailedValidation(t *testing.T) {
	cases := map[string]string{
		"missing name":  `{"email":"regai@ya.ru"}`,
		"invalid email": `{"email":"not-an-email","name":"BlackMamba"}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			_, rec, err := handle(body)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", rec.Code)
			}
		})
	}
}

// Известный баг: тело запроса читается без ограничения размера (нет http.MaxBytesReader),
// поэтому огромный JSON принимается и целиком попадает в память.
func TestHandleBodyRejectsOversizedBody(t *testing.T) {
	if os.Getenv("KNOWN_BUGS") == "" {
		t.Skip("известный баг: нет лимита на размер тела запроса (запуск: KNOWN_BUGS=1)")
	}
	huge := `{"email":"regai@ya.ru","name":"` + strings.Repeat("a", 2<<20) + `"}`
	_, _, err := handle(huge)
	if err == nil {
		t.Fatal("expected request body over 1 MiB to be rejected")
	}
}
