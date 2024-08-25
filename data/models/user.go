package models

type User struct {
	ID        int64  `validate:"required" json:"id"`
	Email     string `validate:"required,email" json:"email" db:"email"`
	Password  string `validate:"required,min=6,max120" json:"password" db:"password"`
	CreatedAt string `json:"createdAt" db:"created_at"`
}

func (User) TableName() string {
	return "users"
}

func (u User) ColumnNames() []string {
	return getColumnNames(u)
}
