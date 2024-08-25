package models

import (
	"database/sql"
	"reflect"
)

type Model interface {
	TableName() string
	ColumnNames() []string
}

// GetValsFromModel returns the field values of a model as a slice of interfaces,
// in the order of the model's column names. It is used for reading off of the model and writing into the db.
// Ensure incoming data has been validated in req handler with ReadJSON before invocation
func GetValsFromModel(m Model) []interface{} {
	val := reflect.ValueOf(m)
	typ := reflect.TypeOf(m)
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

// ScanRowToModel reads the values from a sql.Row into a model.

func ScanRowToModel(m Model, r *sql.Row) error {
	val := reflect.ValueOf(m).Elem()
	typ := val.Type()

	vals := make([]interface{}, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		vals[i] = val.Field(i).Addr().Interface()
	}

	if err := r.Scan(vals...); err != nil {
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
