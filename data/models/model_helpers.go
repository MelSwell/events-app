package models

import (
	"database/sql"
	"fmt"
	"reflect"
)

type Model interface {
	TableName() string
	ColumnNames() []string
	GetID() int64
}

// GetValsFromModel returns the field values of a model as a slice of interfaces,
// in the order of the model's column names. It is used for reading off of the model and writing into the db.
// Ensure incoming data has been validated in req handler with ReadJSON before invocation
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

// ScanRowToModel scans a sql.Row into a Model. The model must be a pointer to a struct implementing the Model interface.
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

func getColumnNames(m Model) []string {
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
