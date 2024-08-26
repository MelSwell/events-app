package models

import "time"

type Event struct {
	ID          int64     `json:"id" db:"id"`
	UserID      int64     `json:"user_id" db:"user_id"`
	Name        string    `validate:"required,min=8,max=100" json:"name" db:"name"`
	Description string    `validate:"required,min=8,max=500" json:"description" db:"description"`
	StartDate   time.Time `validate:"required" json:"start_date" db:"start_date"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

func (Event) TableName() string {
	return "events"
}

func (e Event) ColumnNames() []string {
	return GetColumnNames(e)
}

func (e Event) GetID() int64 {
	return e.ID
}
