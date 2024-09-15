package repository

import (
	"events-app/data/models"
	"log"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/assert"
)

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
