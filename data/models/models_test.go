package models

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

type MockModel struct {
	ID        int    `db:"id"`
	Name      string `db:"name"`
	Email     string `db:"email"`
	CreatedAt string `db:"created_at"`
}

func (m MockModel) TableName() string {
	return "mock_models"
}

func (m MockModel) ColumnNames() []string {
	return getColumnNames(m)
}

func TestGetValsFromModel(t *testing.T) {
	model := MockModel{
		ID:        1,
		Name:      "Test",
		Email:     "example@email.com",
		CreatedAt: "2023-10-01",
	}

	vals := GetValsFromModel(model)
	expectedVals := []interface{}{"Test", "example@email.com"}

	assert.Equal(t, expectedVals, vals)
}

func TestScanRowToModel(t *testing.T) {
	model := &MockModel{}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "name", "email", "created_at"}).
		AddRow(1, "Test", "example@email.com", "2023-10-01")

	mock.ExpectQuery("SELECT \\* FROM mock_models WHERE id = \\?").WillReturnRows(rows)
	row := db.QueryRow("SELECT * FROM mock_models WHERE id = ?", 1)

	err = ScanRowToModel(model, row)
	assert.NoError(t, err)
	assert.Equal(t, 1, model.ID)
	assert.Equal(t, "Test", model.Name)
	assert.Equal(t, "example@email.com", model.Email)
	assert.Equal(t, "2023-10-01", model.CreatedAt)
}
