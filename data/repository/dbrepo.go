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
	Update(m models.Model) error
	Delete(m models.Model) error
	GetModelByID(m models.Model, id int64) (models.Model, error)
	GetUserByID(id int64) (models.User, error)
	GetEventByID(id int64) (models.Event, error)
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

// Create inserts a model into the corresponding db table and returns id of the
// newly created record.
func (sr *SqlRepo) Create(m models.Model) (id int64, err error) {
	vals := models.GetValsFromModel(m)

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING id",
		m.TableName(),
		strings.Join(m.ColumnNames(), ", "),
		placeholders(len(vals)))

	stmt, err := sr.DB.Prepare(query)
	if err != nil {
		return 0, fmt.Errorf("error preparing query: %v", err)
	}
	defer stmt.Close()

	row := stmt.QueryRow(vals...)
	if err := row.Scan(&id); err != nil {
		return 0, fmt.Errorf("error executing query: %v", err)
	}

	return id, nil
}

func (sr *SqlRepo) Update(m models.Model) error {
	columns := m.ColumnNames()

	setClause := make([]string, (len(columns)))
	for i, c := range columns {
		setClause[i] = fmt.Sprintf("%s = $%d", c, i+1)
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = $%d",
		m.TableName(),
		strings.Join(setClause, ", "),
		len(columns)+1)

	stmt, err := sr.DB.Prepare(query)
	if err != nil {
		return fmt.Errorf("error preparing query: %v", err)
	}
	defer stmt.Close()

	vals := models.GetValsFromModel(m)
	vals = append(vals, m.GetID())
	if _, err := stmt.Exec(vals...); err != nil {
		return fmt.Errorf("error executing query: %v", err)
	}
	return nil
}

func (sr *SqlRepo) Delete(m models.Model) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = $1", m.TableName())
	stmt, err := sr.DB.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	if _, err = stmt.Exec(m.GetID()); err != nil {
		return fmt.Errorf("error deleting record: %v", err)
	}
	return nil
}

// GetModelByID retrieves a model from the db by its ID and returns it. The
// model must be passed as a pointer to the desired model type.
func (sr *SqlRepo) GetModelByID(m models.Model, id int64) (models.Model, error) {
	query := fmt.Sprintf("SELECT * FROM %s WHERE id = $1", m.TableName())
	r := sr.DB.QueryRow(query, id)

	if err := models.ScanRowToModel(m, r); err != nil {
		return nil, err
	}
	return m, nil
}

func (sr *SqlRepo) GetUserByID(id int64) (models.User, error) {
	model, err := sr.GetModelByID(&models.User{}, id)
	if err != nil {
		return models.User{}, err
	}

	user, ok := model.(*models.User)
	if !ok {
		return models.User{}, fmt.Errorf("type assertion to User failed")
	}

	return *user, nil
}

func (sr *SqlRepo) GetEventByID(id int64) (models.Event, error) {
	model, err := sr.GetModelByID(&models.Event{}, id)
	if err != nil {
		return models.Event{}, err
	}

	event, ok := model.(*models.Event)
	if !ok {
		return models.Event{}, fmt.Errorf("type assertion to Event failed")
	}

	return *event, nil
}

func placeholders(n int) string {
	ph := make([]string, n)
	for i := 1; i <= n; i++ {
		ph[i-1] = fmt.Sprintf("$%d", i)
	}
	return strings.Join(ph, ", ")
}
