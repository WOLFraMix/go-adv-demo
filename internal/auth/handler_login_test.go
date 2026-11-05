package auth_test

import (
	"bytes"
	"encoding/json"
	"go-adv-demo/internal/auth"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// Хеш bcrypt для пароля "111" (тот же, что в handler_test.go).
const hashOf111 = "$2a$10$0CrlczqbWDqHjPAcI3TSCOx/CQGIHQREUai.Yt.7PcTlpi0VL60PC"

func TestLoginHandlerRejectsWrongPassword(t *testing.T) {
	handler, mock, err := bootstrap()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	mock.ExpectQuery("SELECT.*FROM.*users").WillReturnRows(
		sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "email", "password", "name"}).
			AddRow(1, now, now, nil, "regai@ya.ru", hashOf111, "BlackMamba"))

	data, _ := json.Marshal(&auth.LoginRequest{Email: "regai@ya.ru", Password: "wrong-password"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(data))
	handler.Login()(w, req)

	if w.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Result().StatusCode)
	}
	if strings.Contains(w.Body.String(), "token") {
		t.Fatal("no token must be issued for wrong password")
	}
}

func TestLoginHandlerRejectsMalformedBody(t *testing.T) {
	handler, _, err := bootstrap()
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":`))
	handler.Login()(w, req)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Result().StatusCode)
	}
}

func TestLoginHandlerRejectsInvalidEmailBeforeDB(t *testing.T) {
	handler, mock, err := bootstrap()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT.*FROM.*users").WillReturnRows(sqlmock.NewRows([]string{"id"}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/login",
		strings.NewReader(`{"email":"not-an-email","password":"111"}`))
	handler.Login()(w, req)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Result().StatusCode)
	}
	if mock.ExpectationsWereMet() == nil {
		t.Fatal("validation must reject the request before touching the database")
	}
}
