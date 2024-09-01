package repository

import (
	"database/sql"
	"events-app/data/models"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
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

func handleRecover(name string) {
	if r := recover(); r != nil {
		log.Printf("Test: %s recovered from panic: %v", name, r)
	}
}

func TestMain(m *testing.M) {
	var code int
	defer func() {
		handleRecover("TestMain")
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
	t.Run("Create test User", func(t *testing.T) {
		defer handleRecover(t.Name())

		u := models.User{
			Email:    "hello@example.com",
			Password: "password",
		}
		id, err := testRepo.Create(u)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), id)
	})

	t.Run("Create test Event", func(t *testing.T) {
		defer handleRecover(t.Name())

		e := models.Event{
			UserID:      1,
			Name:        "Test Event",
			Description: "A test event",
			StartDate:   time.Now().Add(time.Hour * 24),
		}
		id, err := testRepo.Create(e)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), id)
	})

	t.Run("Test GetUserByID", func(t *testing.T) {
		defer handleRecover(t.Name())

		u, err := testRepo.GetUserByID(1)
		assert.NoError(t, err)

		assert.Equal(t, "hello@example.com", u.Email)
		assert.Equal(t, int64(1), u.ID)
		assert.NotEmpty(t, u.Password)
		assert.NotEmpty(t, u.CreatedAt)
	})

	t.Run("Test GetEventByID", func(t *testing.T) {
		defer handleRecover(t.Name())

		e, err := testRepo.GetEventByID(1)
		assert.NoError(t, err)

		assert.Equal(t, int64(1), e.ID)
		assert.Equal(t, int64(1), e.UserID)
		assert.Equal(t, "Test Event", e.Name)
		assert.Equal(t, "A test event", e.Description)
		assert.NotEmpty(t, e.StartDate)
		assert.NotEmpty(t, e.CreatedAt)
	})

	t.Run("Test Update", func(t *testing.T) {
		defer handleRecover(t.Name())

		u, err := testRepo.GetUserByID(1)
		assert.NoError(t, err)

		u.Email = "newEmail@example.com"
		err = testRepo.Update(u)
		assert.NoError(t, err)
	})

	t.Run("Test persistence of Update", func(t *testing.T) {
		defer handleRecover(t.Name())

		u, err := testRepo.GetUserByID(1)
		assert.NoError(t, err)

		assert.Equal(t, "newEmail@example.com", u.Email)
	})

	t.Run("Test unique constraint", func(t *testing.T) {
		defer handleRecover(t.Name())

		u := models.User{
			Email:    "newEmail@example.com",
			Password: "password",
		}
		_, err := testRepo.Create(u)
		assert.Error(t, err)
	})

	t.Run("Test Delete", func(t *testing.T) {
		defer handleRecover(t.Name())

		u, err := testRepo.GetEventByID(1)
		assert.NoError(t, err)

		err = testRepo.Delete(u)
		assert.NoError(t, err)
	})

	t.Run("Test persistence of Delete", func(t *testing.T) {
		defer handleRecover(t.Name())

		_, err := testRepo.GetEventByID(1)
		assert.Error(t, err)
	})

	t.Run("Test QueryEvents", func(t *testing.T) {
		defer handleRecover(t.Name())
		seedDBWithEvents(t)

		var tests = []struct {
			name        string
			queryParams map[string]string
			expectedLen int
			expectedErr string
		}{
			{
				name:        "valid query",
				queryParams: map[string]string{"name": "Test Event"},
				expectedLen: 2,
			},
			{
				name:        "simple query",
				queryParams: map[string]string{"name": "Event"},
				expectedLen: 1,
			},
			{
				name:        "no query params",
				queryParams: map[string]string{},
				expectedLen: 10,
			},
			{
				name:        "increase limit",
				queryParams: map[string]string{"limit": "20"},
				expectedLen: 18,
			},
			{
				name:        "invalid model field",
				queryParams: map[string]string{"name": "Test Event", "noSuchThing": "who cares?"},
				expectedErr: "invalid query: invalid query parameter: noSuchThing",
			},
			{
				name:        "should be empty",
				queryParams: map[string]string{"name": "noSuchEvent"},
				expectedLen: 0,
			},
			{
				name:        "test multi-word field name",
				queryParams: map[string]string{"maxAttendees": "25"},
				expectedLen: 1,
			},
			{
				name:        "test multi-word field name; increase limit",
				queryParams: map[string]string{"maxAttendees": "75", "limit": "20"},
				expectedLen: 15,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				defer handleRecover(tt.name)
				events, err := testRepo.QueryEvents(tt.queryParams)

				if tt.expectedErr != "" {
					assert.EqualError(t, err, tt.expectedErr)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tt.expectedLen, len(events))

					switch tt.name {
					case "valid query":
						assert.Equal(t, "At the manor hotel", events[0].Description)
						assert.Equal(t, "A different event with the same name", events[1].Description)
					case "simple query":
						assert.Equal(t, "A different event with a different name", events[0].Description)
					}

				}
			})
		}
	})
}

func seedDBWithEvents(t *testing.T) {
	defer handleRecover("seeding DB")
	log.Println("Seeding DB")

	var events []models.Event
	e1 := models.Event{
		UserID:       1,
		Name:         "Test Event",
		Description:  "At the manor hotel",
		StartDate:    time.Now().Add(time.Hour * 24),
		MaxAttendees: 100,
	}
	e2 := models.Event{
		UserID:       1,
		Name:         "Test Event",
		Description:  "A different event with the same name",
		StartDate:    time.Now().Add(time.Hour * 48),
		MaxAttendees: 50,
	}
	e3 := models.Event{
		UserID:       1,
		Name:         "Event",
		Description:  "A different event with a different name",
		StartDate:    time.Now().Add(time.Hour * 72),
		MaxAttendees: 25,
	}
	events = append(events, e1, e2, e3)

	faker := gofakeit.New(0)
	for i := 0; i < 15; i++ {
		e := models.Event{
			UserID:       1,
			Name:         faker.LoremIpsumSentence(4),
			Description:  faker.LoremIpsumSentence(15),
			StartDate:    faker.FutureDate(),
			MaxAttendees: 75,
		}
		if _, err := testRepo.Create(e); err != nil {
			t.Fatalf("Could not seed DB: %s", err)
		}
	}

	for _, e := range events {
		if _, err := testRepo.Create(e); err != nil {
			t.Fatalf("Could not seed DB: %s", err)
		}
	}
	log.Println("DB Seeded")
}
