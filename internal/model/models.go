package model

import (
	"time"

	"github.com/google/uuid"
)

type Vehicle struct {
	ID          uuid.UUID     `db:"id" json:"id"`
	PlateNumber string        `db:"plate_number" json:"plate_number"`
	LastStatus  VehicleStatus `db:"last_status" json:"last_status"`
}

type VehicleStatus struct {
	Location  [2]float64 `json:"location"`
	Speed     float64    `json:"speed"`
	Timestamp time.Time  `json:"timestamp"`
}

type Trip struct {
	ID        uuid.UUID `db:"id" json:"id"`
	VehicleID uuid.UUID `db:"vehicle_id" json:"vehicle_id"`
	StartTime time.Time `db:"start_time" json:"start_time"`
	EndTime   time.Time `db:"end_time" json:"end_time"`
	Mileage   float64   `db:"mileage" json:"mileage"`
	AvgSpeed  float64   `db:"avg_speed" json:"avg_speed"`
}

type IngestData struct {
	VehicleID uuid.UUID     `json:"vehicle_id"`
	Status    VehicleStatus `json:"status"`
}
