package models

import (
	"database/sql"
	"fmt"
	"reflect"

	"github.com/go-playground/validator"
)

type Model interface {
	TableName() string
	GetID() int64
	EmptySlice() interface{}
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
// extracting values from the model and writing them to the database. Validation
// of the model should be done before use.
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

		if field.Tag.Get("readOnly") == "true" {
			continue
		}

		dbTag := field.Tag.Get("db")
		fieldMap[dbTag] = val.Field(i).Interface()
	}

	columnNames := GetColumnNames(m, true)
	vals := make([]interface{}, len(columnNames))
	for i, cn := range columnNames {
		vals[i] = fieldMap[cn]
	}

	return vals
}

// ScanRowToModel scans a single SQL row into a given model. It takes a model
// and passes a slice of pointers to the model's fields to the sql.Row's Scan
// method. It returns an error if the scan fails or the model is not a pointer.
func ScanRowToModel(m Model, r *sql.Row) error {
	val := reflect.ValueOf(m)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("expected pointer to model, got %T", m)
	}
	val = val.Elem()
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

func ScanRowsToSliceOfModels(m Model, rows *sql.Rows, expectedRows int) (interface{}, error) {
	// Obtain the slice of models using the EmptySlice method, which returns a
	// pointer to an empty slice of the model type as an interface{}
	modelsSlice := m.EmptySlice()

	// Dereference the interface wrapper with Elem(), and make sure we have a slice
	sliceVal := reflect.ValueOf(modelsSlice).Elem()
	if sliceVal.Kind() != reflect.Slice {
		return nil, fmt.Errorf("expected slice, got %s", sliceVal.Kind())
	}

	// Get the type of the model in the slice
	elemType := sliceVal.Type().Elem()

	// We can optimize by setting the initial capacity of the slice to avoid
	// resizing the slice multiple times. We're makng our best guess based on the
	// expected number of rows specified by the caller (e.g. the limit parameter
	// of a URL query).
	initialCapacity := determineInitialCapacity(expectedRows)
	sliceVal.Set(reflect.MakeSlice(sliceVal.Type(), 0, initialCapacity))

	for rows.Next() {
		// Create a new instance of the model type and dereference it
		model := reflect.New(elemType).Elem()

		// Prepare field pointers for scanning
		fieldPtrs := make([]interface{}, model.NumField())
		for i := 0; i < model.NumField(); i++ {
			fieldPtrs[i] = model.Field(i).Addr().Interface()
		}

		// Scan the row into the model's fields
		if err := rows.Scan(fieldPtrs...); err != nil {
			return nil, err
		}

		// Append the new model instance to the slice
		sliceVal.Set(reflect.Append(sliceVal, model))
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return modelsSlice, nil
}

// GetColumnNames returns the model's column names as a slice of strings.
func GetColumnNames(m Model, excludeReadOnlyFields bool) []string {
	val := reflect.ValueOf(m)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	typ := val.Type()
	var columnNames []string

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("db")

		if excludeReadOnlyFields {

			if field.Tag.Get("readOnly") == "true" {
				continue
			}

		}

		columnNames = append(columnNames, tag)
	}
	return columnNames
}

// Returns a map of the model's field tags where key is JSON and value is DB
func MapJsonTagsToDB(m Model) map[string]string {
	val := reflect.ValueOf(m)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	typ := val.Type()
	tagMap := make(map[string]string)

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		jsonTag := field.Tag.Get("json")
		tagMap[jsonTag] = field.Tag.Get("db")
	}
	return tagMap
}

// Helper function to determine the initial capacity based on expected rows
func determineInitialCapacity(expectedRows int) int {
	switch {
	case expectedRows <= 10:
		return 10
	case expectedRows <= 25:
		return 20
	case expectedRows <= 50:
		return 35
	case expectedRows <= 100:
		return 75
	case expectedRows <= 200:
		return 150
	case expectedRows <= 300:
		return 250
	case expectedRows <= 500:
		return 400
	case expectedRows <= 1000:
		return 900
	case expectedRows <= 2000:
		return 1800
	case expectedRows <= 5000:
		return 2500
	default:
		return 5000
	}
}
