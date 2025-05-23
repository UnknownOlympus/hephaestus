package models

// Employee represents an employee entity.
type Employee struct {
	ID       int    `json:"id"`
	FullName string `json:"fullname"`
	Position string `json:"position"`
	Email    string `json:"email"`
	Phone    string `json:"phoneNumber"`
}
