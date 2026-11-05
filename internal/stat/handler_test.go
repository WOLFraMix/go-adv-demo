package stat_test

import (
	"encoding/json"
	"go-adv-demo/configs"
	"go-adv-demo/internal/stat"
	"go-adv-demo/pkg/db"
	"go-adv-demo/pkg/jwt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const testSecret = "stat-handler-test-secret"

type fixture struct {
	t     *testing.T
	mux   *http.ServeMux
	mock  sqlmock.Sqlmock
	repo  *stat.StatRepository
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

	repo := stat.NewStatRepository(&db.Db{DB: gormDB})
	mux := http.NewServeMux()
	stat.NewStatHandler(mux, stat.StatHandlerDeps{
		StatRepository: repo,
		Config:         &configs.Config{Auth: configs.AuthConfig{Secret: testSecret}},
	})

	token, err := jwt.NewJWT(testSecret).Create(jwt.JWTData{Email: "user@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	return &fixture{t: t, mux: mux, mock: mock, repo: repo, token: token}
}

func (f *fixture) do(path, token string) *httptest.ResponseRecorder {
	f.t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)
	return rec
}

func TestGetStatRequiresAuth(t *testing.T) {
	f := newFixture(t)
	rec := f.do("/stat?from=2026-11-01&to=2026-11-30&by=day", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestGetStatRejectsInvalidParams(t *testing.T) {
	cases := map[string]string{
		"missing from": "/stat?to=2026-11-30&by=day",
		"bad to":       "/stat?from=2026-11-01&to=30.11.2026&by=day",
		"unknown by":   "/stat?from=2026-11-01&to=2026-11-30&by=week",
		"missing by":   "/stat?from=2026-11-01&to=2026-11-30",
	}
	for name, path := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			rec := f.do(path, f.token)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", rec.Code)
			}
		})
	}
}

func TestGetStatGroupsByDay(t *testing.T) {
	f := newFixture(t)
	f.mock.ExpectQuery(`SELECT to_char\(date, 'YYYY-MM-DD'\) as period, sum\(clicks\) FROM "stats"`).
		WillReturnRows(sqlmock.NewRows([]string{"period", "sum"}).AddRow("2026-11-05", 3))

	rec := f.do("/stat?from=2026-11-01&to=2026-11-30&by=day", f.token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", rec.Code, rec.Body.String())
	}

	var got []stat.GetStatResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Period != "2026-11-05" || got[0].Sum != 3 {
		t.Fatalf("unexpected response: %+v", got)
	}
	if err := f.mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetStatGroupsByMonth(t *testing.T) {
	f := newFixture(t)
	f.mock.ExpectQuery(`SELECT to_char\(date, 'YYYY-MM'\) as period, sum\(clicks\) FROM "stats"`).
		WillReturnRows(sqlmock.NewRows([]string{"period", "sum"}).AddRow("2026-11", 42))

	rec := f.do("/stat?from=2026-11-01&to=2026-11-30&by=month", f.token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", rec.Code, rec.Body.String())
	}
	if err := f.mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
