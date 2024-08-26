package models

import (
	"database/sql"
	"fmt"
	"reflect"

	"github.com/go-playground/validator"
)

type Model interface {
	TableName() string
	ColumnNames() []string
	GetID() int64
}

// go-playground/validator suggests using a single instance of the validator, I
// may end up needing to instantiate this higher up in the data flow? Seems fine
// for now Alternatively, expose it as a const here and import in the main
// package and attach to the app struct
var validate = validator.New()

// ValidateModel validates a model using the go-playground/validator package. It
// returns an error if the provided argument does not implement the Model
// interface.
func ValidateModel(model interface{}) error {
	m, ok := model.(Model)
	if !ok {
		return fmt.Errorf("expected model, got %T", m)
	}

	if err := validate.Struct(m); err != nil {
		return err
	}
	return nil
}

// GetValsFromModel returns the field values of a model as a slice of
// interfaces, in the order of the model's column names. It is used for
// extracting values from the model and writing them to the database. Ensure the
// model has been validated using ValidateModel before calling.
func GetValsFromModel(m Model) []interface{} {
	val := reflect.ValueOf(m)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	typ := val.Type()
	numFields := val.NumField()

	fieldMap := make(map[string]interface{})
	for i := 0; i < numFields; i++ {
		field := typ.Field(i)
		// skip default fields managed by the DB
		if field.Name == "ID" || field.Name == "CreatedAt" {
			continue
		}
		dbTag := field.Tag.Get("db")
		fieldMap[dbTag] = val.Field(i).Interface()
	}

	columnNames := m.ColumnNames()
	vals := make([]interface{}, len(columnNames))
	for i, cn := range columnNames {
		vals[i] = fieldMap[cn]
	}

	return vals
}

// ScanRowToModel scans a single SQL row into a given model.
// It takes a pointer to a model and passes a slice of pointers
// to the model's fields to the sql.Row's Scan method.
// It returns an error if the scan fails or the model is not a pointer.
func ScanRowToModel(m Model, r *sql.Row) error {
	val := reflect.ValueOf(m)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	} else {
		return fmt.Errorf("expected pointer to struct, got %s", val.Kind())
	}
	typ := val.Type()

	fieldPtrs := make([]interface{}, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		fieldPtrs[i] = val.Field(i).Addr().Interface()
	}

	if err := r.Scan(fieldPtrs...); err != nil {
		return err
	}
	return nil
}

// GetColumnNames returns the field names of a model as a slice of db formatted strings.
// The order of the returned slice is the order in which the fields are defined in the model.
// Ensure model fields are defined in the order that they are defined in the db schema, and a corresponding db tag is set.
func GetColumnNames(m Model) []string {
	typ := reflect.TypeOf(m)
	var columnNames []string

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("db")
		// skip default fields managed by the DB
		if tag == "id" || tag == "created_at" {
			continue
		}
		columnNames = append(columnNames, tag)
	}
	return columnNames
}
