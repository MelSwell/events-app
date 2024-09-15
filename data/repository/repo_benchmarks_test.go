package repository

import (
	"events-app/data/models"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
)

func SeedDBforBenchmark(b *testing.B) {
	defer handleRecover("seeding DB")
	u := models.User{
		Email:    gofakeit.Email(),
		Password: "password",
	}
	_, err := testRepo.Create(u)
	if err != nil {
		b.Fatalf("Could not seed DB: %s", err)
	}

	for i := 0; i < 1000; i++ {
		e := models.Event{
			UserID:       1,
			Name:         gofakeit.LoremIpsumSentence(4),
			Description:  gofakeit.LoremIpsumSentence(15),
			StartDate:    gofakeit.FutureDate(),
			MaxAttendees: 75,
		}
		if _, err := testRepo.Create(e); err != nil {
			b.Fatalf("Could not seed DB: %s", err)
		}
	}
}

func BenchmarkCreate(b *testing.B) {
	defer handleRecover("BenchmarkCreate")

	u := models.User{
		Email:    gofakeit.Email(),
		Password: "password",
	}
	_, err := testRepo.Create(u)
	if err != nil {
		b.Fatalf("Could not seed DB: %s", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e := models.Event{
			UserID:       1,
			Name:         gofakeit.LoremIpsumSentence(4),
			Description:  gofakeit.LoremIpsumSentence(15),
			StartDate:    gofakeit.FutureDate(),
			MaxAttendees: 75,
		}
		if _, err := testRepo.Create(e); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkQueryEvents_Limit1000(b *testing.B) {
	defer handleRecover("BenchmarkQueryModel_1000")

	SeedDBforBenchmark(b)
	queryParams := map[string]string{"limit": "1000"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := testRepo.QueryEvents(queryParams)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkQueryEvents_Limit10(b *testing.B) {
	defer handleRecover("BenchmarkQueryModel_10")

	SeedDBforBenchmark(b)
	queryParams := map[string]string{"limit": "10"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := testRepo.QueryEvents(queryParams)
		if err != nil {
			b.Fatal(err)
		}
	}
}
func BenchmarkQueryEvents_Limit500(b *testing.B) {
	defer handleRecover("BenchmarkQueryModel_500")

	SeedDBforBenchmark(b)
	queryParams := map[string]string{"limit": "500"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := testRepo.QueryEvents(queryParams)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkQueryEvents_Limit100(b *testing.B) {
	defer handleRecover("BenchmarkQueryModel_100")

	SeedDBforBenchmark(b)
	queryParams := map[string]string{"limit": "100"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := testRepo.QueryEvents(queryParams)
		if err != nil {
			b.Fatal(err)
		}
	}
}
func BenchmarkQueryEvents_Limit2000(b *testing.B) {
	defer handleRecover("BenchmarkQueryModel_2000")

	SeedDBforBenchmark(b)
	queryParams := map[string]string{"limit": "2000"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := testRepo.QueryEvents(queryParams)
		if err != nil {
			b.Fatal(err)
		}
	}
}
