package link_test

import (
	"encoding/json"
	"go-adv-demo/configs"
	"go-adv-demo/internal/link"
	"go-adv-demo/pkg/db"
	"go-adv-demo/pkg/event"
	"go-adv-demo/pkg/jwt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const testSecret = "link-handler-test-secret"

var linkColumns = []string{"id", "created_at", "updated_at", "deleted_at", "url", "hash"}

type fixture struct {
	t     *testing.T
	mux   *http.ServeMux
	mock  sqlmock.Sqlmock
	bus   *event.EventBus
	token string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	bus := event.NewEventBus()
	mux := http.NewServeMux()
	link.NewLinkHandler(mux, link.LinkHandlerDeps{
		LinkRepository: link.NewLinkRepository(&db.Db{DB: gormDB}),
		EventBus:       bus,
		Config:         &configs.Config{Auth: configs.AuthConfig{Secret: testSecret}},
	})

	token, err := jwt.NewJWT(testSecret).Create(jwt.JWTData{Email: "user@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	return &fixture{t: t, mux: mux, mock: mock, bus: bus, token: token}
}

func (f *fixture) do(method, path, body, token string) *httptest.ResponseRecorder {
	f.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)
	return rec
}

func (f *fixture) expectStatus(rec *httptest.ResponseRecorder, want int) {
	f.t.Helper()
	if rec.Code != want {
		f.t.Fatalf("expected status %d, got %d, body: %s", want, rec.Code, rec.Body.String())
	}
}

func (f *fixture) expectAllMet() {
	f.t.Helper()
	if err := f.mock.ExpectationsWereMet(); err != nil {
		f.t.Fatal(err)
	}
}

// expectNoDBCall проверяет, что запрос отклонён до обращения к БД:
// заранее заведённое ожидание должно остаться невыполненным.
func (f *fixture) expectNoDBCall() {
	f.t.Helper()
	if f.mock.ExpectationsWereMet() == nil {
		f.t.Fatal("expected request to be rejected before touching the database")
	}
}

func (f *fixture) expectCreateSuccess() {
	f.mock.ExpectQuery(`SELECT .* FROM "links"`).WillReturnRows(sqlmock.NewRows(linkColumns))
	f.mock.ExpectBegin()
	f.mock.ExpectQuery(`INSERT INTO "links"`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	f.mock.ExpectCommit()
}

// ---------- POST /link ----------

func TestCreateLinkRequiresAuth(t *testing.T) {
	f := newFixture(t)
	rec := f.do(http.MethodPost, "/link", `{"url":"https://example.com"}`, "")
	f.expectStatus(rec, http.StatusUnauthorized)
}

func TestCreateLinkSuccess(t *testing.T) {
	f := newFixture(t)
	f.expectCreateSuccess()

	rec := f.do(http.MethodPost, "/link", `{"url":"https://example.com"}`, f.token)
	f.expectStatus(rec, http.StatusCreated)

	var created link.Link
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Url != "https://example.com" {
		t.Fatalf("expected url to be echoed back, got %q", created.Url)
	}
	if len(created.Hash) != 10 {
		t.Fatalf("expected 10-char hash, got %q", created.Hash)
	}
	f.expectAllMet()
}

func TestCreateLinkRejectsInvalidURL(t *testing.T) {
	f := newFixture(t)
	f.mock.ExpectQuery(`SELECT .* FROM "links"`).WillReturnRows(sqlmock.NewRows(linkColumns))

	rec := f.do(http.MethodPost, "/link", `{"url":"not a url"}`, f.token)
	f.expectStatus(rec, http.StatusBadRequest)
	f.expectNoDBCall()
}

func TestCreateLinkRejectsMalformedJSON(t *testing.T) {
	f := newFixture(t)
	f.mock.ExpectQuery(`SELECT .* FROM "links"`).WillReturnRows(sqlmock.NewRows(linkColumns))

	rec := f.do(http.MethodPost, "/link", `{"url":`, f.token)
	f.expectStatus(rec, http.StatusBadRequest)
	f.expectNoDBCall()
}

// Известный баг: validator tag "url" принимает javascript: и data: схемы,
// а GET /{hash} потом отдаёт такой URL в Location.
func TestCreateLinkRejectsDangerousSchemes(t *testing.T) {
	if os.Getenv("KNOWN_BUGS") == "" {
		t.Skip("известный баг: url-валидация пропускает javascript:/data: (запуск: KNOWN_BUGS=1)")
	}
	for _, url := range []string{"javascript:alert(1)", "data:text/html,<script>alert(1)</script>"} {
		t.Run(url, func(t *testing.T) {
			f := newFixture(t)
			f.expectCreateSuccess()

			rec := f.do(http.MethodPost, "/link", `{"url":"`+url+`"}`, f.token)
			f.expectStatus(rec, http.StatusBadRequest)
		})
	}
}

// ---------- GET /{hash} ----------

func TestGoToRedirectsAndPublishesVisit(t *testing.T) {
	f := newFixture(t)
	now := time.Now()
	f.mock.ExpectQuery(`SELECT .* FROM "links"`).WillReturnRows(
		sqlmock.NewRows(linkColumns).AddRow(7, now, now, nil, "https://example.com/page", "abcdefghij"))

	rec := f.do(http.MethodGet, "/abcdefghij", "", "")
	f.expectStatus(rec, http.StatusTemporaryRedirect)
	if loc := rec.Header().Get("Location"); loc != "https://example.com/page" {
		t.Fatalf("expected redirect to target url, got %q", loc)
	}

	select {
	case ev := <-f.bus.Subscribe():
		if ev.Type != event.LinkVisitedEvent {
			t.Fatalf("expected %s event, got %s", event.LinkVisitedEvent, ev.Type)
		}
		if ev.Data != uint(7) {
			t.Fatalf("expected link id 7 in event, got %v", ev.Data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("visit event was not published")
	}
}

func TestGoToUnknownHashReturnsNotFound(t *testing.T) {
	f := newFixture(t)
	f.mock.ExpectQuery(`SELECT .* FROM "links"`).WillReturnRows(sqlmock.NewRows(linkColumns))

	rec := f.do(http.MethodGet, "/doesnotexist", "", "")
	f.expectStatus(rec, http.StatusNotFound)
}

// ---------- GET /link ----------

func TestGetLinksRequiresAuth(t *testing.T) {
	f := newFixture(t)
	rec := f.do(http.MethodGet, "/link?limit=10&offset=0", "", "")
	f.expectStatus(rec, http.StatusUnauthorized)
}

func TestGetLinksRejectsInvalidPagination(t *testing.T) {
	cases := map[string]string{
		"missing limit":  "/link?offset=0",
		"non-numeric":    "/link?limit=abc&offset=0",
		"missing offset": "/link?limit=10",
	}
	for name, path := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			rec := f.do(http.MethodGet, path, "", f.token)
			f.expectStatus(rec, http.StatusBadRequest)
		})
	}
}

func TestGetLinksReturnsPageAndCount(t *testing.T) {
	f := newFixture(t)
	now := time.Now()
	f.mock.ExpectQuery(`SELECT .* FROM "links"`).WillReturnRows(
		sqlmock.NewRows(linkColumns).AddRow(1, now, now, nil, "https://example.com", "abcdefghij"))
	f.mock.ExpectQuery(`SELECT count\(\*\) FROM "links"`).WillReturnRows(
		sqlmock.NewRows([]string{"count"}).AddRow(1))

	rec := f.do(http.MethodGet, "/link?limit=10&offset=0", "", f.token)
	f.expectStatus(rec, http.StatusOK)

	var resp link.GetLinksResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Links) != 1 || resp.Count != 1 {
		t.Fatalf("expected 1 link and count 1, got %d links, count %d", len(resp.Links), resp.Count)
	}
	f.expectAllMet()
}

// Известный баг: limit=-1 у GORM означает «без LIMIT», т.е. отдаётся вся таблица.
func TestGetLinksRejectsNegativeLimit(t *testing.T) {
	if os.Getenv("KNOWN_BUGS") == "" {
		t.Skip("известный баг: отрицательный limit снимает LIMIT (запуск: KNOWN_BUGS=1)")
	}
	f := newFixture(t)
	f.mock.ExpectQuery(`SELECT .* FROM "links"`).WillReturnRows(sqlmock.NewRows(linkColumns))
	f.mock.ExpectQuery(`SELECT count\(\*\) FROM "links"`).WillReturnRows(
		sqlmock.NewRows([]string{"count"}).AddRow(0))

	rec := f.do(http.MethodGet, "/link?limit=-1&offset=0", "", f.token)
	f.expectStatus(rec, http.StatusBadRequest)
}

// ---------- PATCH /link/{id} ----------

func TestUpdateLinkRequiresAuth(t *testing.T) {
	f := newFixture(t)
	rec := f.do(http.MethodPatch, "/link/1", `{"url":"https://example.com"}`, "")
	f.expectStatus(rec, http.StatusUnauthorized)
}

func TestUpdateLinkRejectsNonNumericID(t *testing.T) {
	f := newFixture(t)
	rec := f.do(http.MethodPatch, "/link/abc", `{"url":"https://example.com"}`, f.token)
	f.expectStatus(rec, http.StatusBadRequest)
}

func TestUpdateLinkSuccess(t *testing.T) {
	f := newFixture(t)
	now := time.Now()
	f.mock.ExpectBegin()
	f.mock.ExpectQuery(`UPDATE "links"`).WillReturnRows(
		sqlmock.NewRows(linkColumns).AddRow(1, now, now, nil, "https://new.example.com", "newhash"))
	f.mock.ExpectCommit()

	rec := f.do(http.MethodPatch, "/link/1", `{"url":"https://new.example.com","hash":"newhash"}`, f.token)
	f.expectStatus(rec, http.StatusOK)

	var updated link.Link
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Url != "https://new.example.com" || updated.Hash != "newhash" {
		t.Fatalf("unexpected response: %+v", updated)
	}
	f.expectAllMet()
}

// Известный баг: UPDATE без затронутых строк возвращает 200 с пустой ссылкой.
func TestUpdateMissingLinkReturnsNotFound(t *testing.T) {
	if os.Getenv("KNOWN_BUGS") == "" {
		t.Skip("известный баг: PATCH несуществующего id отвечает 200 (запуск: KNOWN_BUGS=1)")
	}
	f := newFixture(t)
	f.mock.ExpectBegin()
	f.mock.ExpectQuery(`UPDATE "links"`).WillReturnRows(sqlmock.NewRows(linkColumns))
	f.mock.ExpectCommit()

	rec := f.do(http.MethodPatch, "/link/999", `{"url":"https://example.com"}`, f.token)
	f.expectStatus(rec, http.StatusNotFound)
}

// ---------- DELETE /link/{id} ----------

func TestDeleteLinkRequiresAuth(t *testing.T) {
	f := newFixture(t)
	rec := f.do(http.MethodDelete, "/link/1", "", "")
	f.expectStatus(rec, http.StatusUnauthorized)
}

func TestDeleteLinkNotFound(t *testing.T) {
	f := newFixture(t)
	f.mock.ExpectQuery(`SELECT .* FROM "links"`).WillReturnRows(sqlmock.NewRows(linkColumns))

	rec := f.do(http.MethodDelete, "/link/42", "", f.token)
	f.expectStatus(rec, http.StatusNotFound)
}

func TestDeleteLinkSuccess(t *testing.T) {
	f := newFixture(t)
	now := time.Now()
	f.mock.ExpectQuery(`SELECT .* FROM "links"`).WillReturnRows(
		sqlmock.NewRows(linkColumns).AddRow(1, now, now, nil, "https://example.com", "abcdefghij"))
	f.mock.ExpectBegin()
	f.mock.ExpectExec(`UPDATE "links" SET "deleted_at"`).WillReturnResult(sqlmock.NewResult(0, 1))
	f.mock.ExpectCommit()

	rec := f.do(http.MethodDelete, "/link/1", "", f.token)
	f.expectStatus(rec, http.StatusNoContent)
	f.expectAllMet()
}
