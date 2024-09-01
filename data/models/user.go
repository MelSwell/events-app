package models

import "time"

type User struct {
	ID        int64     `json:"id" db:"id" readOnly:"true"`
	Email     string    `validate:"required,email" json:"email" db:"email"`
	Password  string    `validate:"min=6,max=120" json:"password" db:"password"`
	CreatedAt time.Time `json:"createdAt" db:"created_at" readOnly:"true"`
}

func (User) TableName() string {
	return "users"
}

func (u User) GetID() int64 {
	return u.ID
}

func (u User) EmptySlice() interface{} {
	return &[]User{}
}
