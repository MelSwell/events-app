package models

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestGetColumnNames(t *testing.T) {
	tests := []struct {
		name                  string
		model                 Model
		excludeReadOnlyFields bool
		expectedOutput        []string
	}{
		{
			"User; exclude read only fields",
			User{}, true, []string{
				"email",
				"password",
			},
		},
		{
			"User; include read only fields",
			User{}, false, []string{
				"id",
				"email",
				"password",
				"created_at",
			},
		},
		{
			"Event; exclude read only fields",
			Event{}, true, []string{
				"user_id",
				"name",
				"description",
				"start_date",
				"max_attendees",
			},
		},
		{
			"Event; include read only fields",
			Event{}, false, []string{
				"id",
				"user_id",
				"name",
				"description",
				"start_date",
				"created_at",
				"max_attendees",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			columns := GetColumnNames(tt.model, tt.excludeReadOnlyFields)
			assert.Equal(t, tt.expectedOutput, columns)
		})
	}
}

func TestMapJsonTagsToDB(t *testing.T) {
	tests := []struct {
		name           string
		model          Model
		expectedOutput map[string]string
	}{
		{
			"User", User{},
			map[string]string{
				"id":        "id",
				"email":     "email",
				"password":  "password",
				"createdAt": "created_at",
			},
		},
		{
			"Event", Event{},
			map[string]string{
				"id":           "id",
				"userId":       "user_id",
				"name":         "name",
				"description":  "description",
				"startDate":    "start_date",
				"createdAt":    "created_at",
				"maxAttendees": "max_attendees",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mappings := MapJsonTagsToDB(tt.model)
			assert.Equal(t, tt.expectedOutput, mappings)
		})
	}
}

type MockModel struct {
	ID        int64     `db:"id" readOnly:"true"`
	Name      string    `validate:"required" db:"name"`
	Email     string    `validate:"email" db:"email"`
	CreatedAt time.Time `db:"created_at" readOnly:"true"`
}

func (m MockModel) TableName() string {
	return "mock_models"
}

func (m MockModel) GetID() int64 {
	return m.ID
}

func (m MockModel) EmptySlice() interface{} {
	return &[]MockModel{}
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

func TestScanToModel(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	t.Run("Test scan row to model", func(t *testing.T) {
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
		assert.WithinDuration(t, time.Now(), model.CreatedAt, time.Second)
	})

	t.Run("Test scan rows to slice of models", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "name", "email", "created_at"}).
			AddRow(1, "Test User", "test@example.com", time.Now()).
			AddRow(2, "Another User", "another@example.com", time.Now())

		mock.ExpectQuery("SELECT \\* FROM mock_models").WillReturnRows(rows)

		query := "SELECT * FROM mock_models"
		sqlRows, err := db.Query(query)
		if err != nil {
			t.Fatalf("an error '%s' was not expected when querying the database", err)
		}
		defer sqlRows.Close()

		model := MockModel{}
		results, err := ScanRowsToSliceOfModels(model, sqlRows, 2)
		if err != nil {
			t.Fatalf("an error '%s' was not expected when scanning rows to slice of models", err)
		}

		modelsSlice, ok := results.(*[]MockModel)
		if !ok {
			t.Fatalf("expected *[]MockModel, got %T", results)
		}

		assert.Equal(t, 2, len(*modelsSlice))
		assert.Equal(t, int64(1), (*modelsSlice)[0].ID)
		assert.Equal(t, "Test User", (*modelsSlice)[0].Name)
		assert.Equal(t, "test@example.com", (*modelsSlice)[0].Email)
		assert.Equal(t, int64(2), (*modelsSlice)[1].ID)
		assert.Equal(t, "Another User", (*modelsSlice)[1].Name)
		assert.Equal(t, "another@example.com", (*modelsSlice)[1].Email)
	})
}
