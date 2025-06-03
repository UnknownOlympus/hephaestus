package models

import "time"

type Task struct {
	ID            int       `json:"id"`
	Type          string    `json:"type"`
	CreatedAt     time.Time `json:"createdAt"`
	ClosedAt      time.Time `json:"closedAt"`
	Description   string    `json:"description"`
	Address       string    `json:"address"`
	CustomerName  string    `json:"customerName"`
	CustomerLogin string    `json:"customerLogin"`
	Comments      []string  `json:"comments"`
	Executors     []string  `json:"executors"`
}
