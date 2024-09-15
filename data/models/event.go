package models

import "time"

type Event struct {
	ID           int64     `json:"id" db:"id" readOnly:"true"`
	UserID       int64     `json:"userId" db:"user_id"`
	Name         string    `validate:"required,min=8,max=100" json:"name" db:"name"`
	Description  string    `validate:"required,min=8,max=500" json:"description" db:"description"`
	StartDate    time.Time `validate:"required" json:"startDate" db:"start_date"`
	CreatedAt    time.Time `json:"createdAt" db:"created_at" readOnly:"true"`
	MaxAttendees int       `json:"maxAttendees" db:"max_attendees"`
}

func (Event) TableName() string {
	return "events"
}

func (Event) EmptySlice() interface{} {
	return &[]Event{}
}

func (e Event) GetID() int64 {
	return e.ID
}
