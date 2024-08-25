package repository

import (
	"database/sql"
	"events-app/data/models"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/pgx"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

type DBRepo interface {
	Connection() *sql.DB
	RunMigrations(dbName string) error
	Create(m models.Model) (id int64, err error)
	GetModelByID(m models.Model, id int64) (models.Model, error)
	Update(m models.Model, id int64) error
	Delete(m models.Model, id int64) error
}

type SqlRepo struct {
	DB *sql.DB
}

func (sr *SqlRepo) Connection() *sql.DB {
	return sr.DB
}

func (sr *SqlRepo) RunMigrations(dbName string) error {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("failed to get current file path")
	}

	dir := filepath.Dir(filename)
	migrationsDir := filepath.Join(dir, "../migrations")
	// Convert backslashes to forward slashes for Windows compatibility
	migrationsDir = strings.ReplaceAll(migrationsDir, "\\", "/")

	log.Printf("Resolved migrations directory: %s", migrationsDir)

	driver, err := pgx.WithInstance(sr.DB, &pgx.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://"+migrationsDir, dbName, driver)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %v", err)
	}

	log.Println("Migrations complete")
	return nil
}

// Create inserts a model into the corresponding db table and
// returns the ID of the newly created record.
// For immediate access of new record, pass ID to GetModelByID
func (sr *SqlRepo) Create(m models.Model) (id int64, err error) {
	vals := models.GetValsFromModel(m)

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		m.TableName(),
		strings.Join(m.ColumnNames(), ", "),
		placeholders(len(vals)))

	stmt, err := sr.DB.Prepare(query)
	if err != nil {
		return 0, fmt.Errorf("error preparing query: %v", err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(vals...)
	if err != nil {
		return 0, fmt.Errorf("error executing query: %v", err)
	}

	return res.LastInsertId()
}

func (sr *SqlRepo) Update(m models.Model, id int64) error {
	columns := m.ColumnNames()

	setClause := make([]string, (len(columns)))
	for i, c := range columns {
		setClause[i] = fmt.Sprintf("%s = ?", c)
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?",
		m.TableName(),
		strings.Join(setClause, ", "))

	stmt, err := sr.DB.Prepare(query)
	if err != nil {
		return fmt.Errorf("error preparing query: %v", err)
	}
	defer stmt.Close()

	vals := models.GetValsFromModel(m)
	vals = append(vals, id)
	if _, err := stmt.Exec(vals...); err != nil {
		return fmt.Errorf("error executing query: %v", err)
	}
	return nil
}

func (sr *SqlRepo) Delete(m models.Model, id int64) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", m.TableName())
	stmt, err := sr.DB.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	if _, err = stmt.Exec(id); err != nil {
		return fmt.Errorf("error deleting record: %v", err)
	}
	return nil
}

func (sr *SqlRepo) GetModelByID(m models.Model, id int64) (models.Model, error) {
	query := fmt.Sprintf("SELECT * FROM %s WHERE id = ?", m.TableName())
	r := sr.DB.QueryRow(query, id)

	if err := models.ScanRowToModel(m, r); err != nil {
		return nil, err
	}
	return m, nil
}

func placeholders(n int) string {
	ph := make([]string, n)
	for i := 0; i < n; i++ {
		ph[i] = "?"
	}
	return strings.Join(ph, ", ")
}
