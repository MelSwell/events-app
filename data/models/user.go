package models

import "time"

type User struct {
	ID        int64     `json:"id" db:"id"`
	Email     string    `validate:"required,email" json:"email" db:"email"`
	Password  string    `validate:"min=6,max=120" json:"password" db:"password"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}

func (User) TableName() string {
	return "users"
}

func (u User) ColumnNames() []string {
	return GetColumnNames(u)
}

func (u User) GetID() int64 {
	return u.ID
}
