package models

import (
	"time"

	pb "github.com/UnknownOlympus/olympus-protos/gen/go/scraper/olympus"
)

type Task struct {
	ID          int            `json:"id"`
	Type        string         `json:"type"`
	CreatedAt   time.Time      `json:"createdAt"`
	ClosedAt    time.Time      `json:"closedAt"`
	Description string         `json:"description"`
	Address     string         `json:"address"`
	Customers   []*pb.Customer `json:"customers"`
	Comments    []string       `json:"comments"`
	Executors   []string       `json:"executors"`
	IsClosed    bool           `json:"is_closed"`
}
