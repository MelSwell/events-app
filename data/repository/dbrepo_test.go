package repository

import (
	"database/sql"
	"events-app/data/models"
	"fmt"
	"log"
	"os"
	"testing"

	_ "github.com/jackc/pgconn"
	_ "github.com/jackc/pgx/v4"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/assert"
)

var (
	host     = "localhost"
	user     = "user"
	password = "password"
	dbname   = "test_db"
	port     = "5435"
	dsn      = "host=%s port=%s user=%s password=%s dbname=%s sslmode=disable"
)

var resource *dockertest.Resource
var pool *dockertest.Pool
var testDB *sql.DB
var testRepo DBRepo

func cleanup() {
	log.Println("cleaning up")
	if resource != nil {
		log.Println("Purging resource")
		if err := pool.Purge(resource); err != nil {
			log.Printf("Could not purge resource: %s", err)
		}
	}
	if testDB != nil {
		log.Println("Closing testDB")
		if err := testDB.Close(); err != nil {
			log.Printf("Could not close testDB: %s", err)
		}
	}
}

func handleRecover() {
	if r := recover(); r != nil {
		log.Printf("Recovered from panic: %v", r)
	}
}

func TestMain(m *testing.M) {
	var code int
	defer func() {
		handleRecover()
		cleanup()
		log.Println("Exiting TestMain")
		os.Exit(code)
	}()

	p, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}
	pool = p

	opts := dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "15",
		Env: []string{
			"POSTGRES_USER=" + user,
			"POSTGRES_PASSWORD=" + password,
			"POSTGRES_DB=" + dbname,
		},
		ExposedPorts: []string{"5432"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"5432": {
				{HostIP: "", HostPort: port},
			},
		},
	}

	if resource, err = pool.RunWithOptions(&opts, func(conf *docker.HostConfig) {
		conf.AutoRemove = true
	}); err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	if err := pool.Retry(func() error {
		var err error
		testDB, err = sql.Open("pgx", fmt.Sprintf(dsn, host, port, user, password, dbname))
		if err != nil {
			log.Println("Error:", err)
			return err
		}
		return testDB.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	testRepo = &SqlRepo{DB: testDB}
	if err = testRepo.RunMigrations("test_db"); err != nil {
		log.Fatal(err.Error())
	}

	log.Println("Running tests")
	code = m.Run()
}

func TestDBRepo(t *testing.T) {

	t.Run("Test Create() with a User", func(t *testing.T) {
		defer handleRecover()

		u := models.User{
			Email:    "hello@example.com",
			Password: "password",
		}
		id, err := testRepo.Create(u)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), id)
	})

	t.Run("Test GetModelByID with new User", func(t *testing.T) {
		defer handleRecover()

		var u models.User
		model, err := testRepo.GetModelByID(u, 1)

		u, ok := model.(models.User)
		if !ok {
			t.Fatalf("Expected User, got %T", u)
		}
		assert.NoError(t, err)

		assert.Equal(t, "hello@example.com", u.Email)
		assert.Equal(t, int64(1), u.ID)
		assert.NotEmpty(t, u.Password)
		assert.NotEmpty(t, u.CreatedAt)
	})
}
