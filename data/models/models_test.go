package models

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

type MockModel struct {
	ID        int64     `db:"id"`
	Name      string    `validate:"required" db:"name"`
	Email     string    `validate:"email" db:"email"`
	CreatedAt time.Time `db:"created_at"`
}

func (m MockModel) TableName() string {
	return "mock_models"
}

func (m MockModel) ColumnNames() []string {
	return GetColumnNames(m)
}

func (m MockModel) GetID() int64 {
	return m.ID
}

func TestValidateModel(t *testing.T) {
	tests := []struct {
		name string
		data interface{}
	}{
		{"Valid model", MockModel{1, "Test", "hello@example.com", time.Now()}},
		{"Missing required field", MockModel{1, "", "hello@example.com", time.Now()}},
		{"Invalid field", MockModel{1, "Test", "hello@example", time.Now()}},
		{"Does not implement Model interface", struct{ Name string }{"Test"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateModel(tt.data)
			if tt.name == "Valid model" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}

			if _, ok := tt.data.(Model); ok {
				if !ok {
					assert.Error(t, err)
				}
			}
		})
	}
}

func TestGetValsFromModel(t *testing.T) {
	tests := []struct {
		name  string
		model MockModel
	}{
		{"Fields in correct order", MockModel{1, "Test", "example@email.com", time.Now()}},
		{"Fields out of order", MockModel{Email: "another@example.com", Name: "Test2", ID: 2, CreatedAt: time.Now()}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vals := GetValsFromModel(tt.model)
			expectedVals := []interface{}{tt.model.Name, tt.model.Email}
			assert.Equal(t, expectedVals, vals)
		})
	}
}

func TestScanRowToModel(t *testing.T) {

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "name", "email", "created_at"}).
		AddRow(1, "Test", "example@email.com", time.Now())

	mock.ExpectQuery("SELECT \\* FROM mock_models WHERE id = \\?").WillReturnRows(rows)
	row := db.QueryRow("SELECT * FROM mock_models WHERE id = ?", 1)

	// Function under test
	model := &MockModel{}
	err = ScanRowToModel(model, row)

	assert.NoError(t, err)
	assert.Equal(t, int64(1), model.ID)
	assert.Equal(t, "Test", model.Name)
	assert.Equal(t, "example@email.com", model.Email)
	assert.Equal(t, time.Now(), model.CreatedAt)
}
