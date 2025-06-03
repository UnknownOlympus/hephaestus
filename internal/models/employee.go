package models

// Employee represents an employee entity.
type Employee struct {
	ID        int    `json:"id"`
	FullName  string `json:"fullname"`
	ShortName string `json:"shortname"`
	Position  string `json:"position"`
	Email     string `json:"email"`
	Phone     string `json:"phoneNumber"`
}
