package stat_test

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

var statColumns = []string{"id", "created_at", "updated_at", "deleted_at", "link_id", "clicks", "date"}

func TestAddClickCreatesStatForNewDay(t *testing.T) {
	f := newFixture(t)
	f.mock.ExpectQuery(`SELECT .* FROM "stats"`).WillReturnRows(sqlmock.NewRows(statColumns))
	f.mock.ExpectBegin()
	f.mock.ExpectQuery(`INSERT INTO "stats"`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	f.mock.ExpectCommit()

	f.repo.AddClick(1)

	if err := f.mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAddClickIncrementsExistingStatForDay(t *testing.T) {
	f := newFixture(t)
	now := time.Now()
	f.mock.ExpectQuery(`SELECT .* FROM "stats"`).WillReturnRows(
		sqlmock.NewRows(statColumns).AddRow(3, now, now, nil, 1, 4, now))
	f.mock.ExpectBegin()
	// clicks 4 -> 5, остальные поля перезаписываются Save()
	f.mock.ExpectExec(`UPDATE "stats"`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), nil, 1, 5, sqlmock.AnyArg(), 3).
		WillReturnResult(sqlmock.NewResult(0, 1))
	f.mock.ExpectCommit()

	f.repo.AddClick(1)

	if err := f.mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
