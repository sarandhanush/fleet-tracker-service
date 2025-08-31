package service

import (
	"context"
	"time"

	"fleet-tracker-service/internal/model"
)

type IngestPayload struct {
	VehicleID   string                 `json:"vehicle_id"`
	PlateNumber string                 `json:"plate_number"`
	Status      map[string]interface{} `json:"status"`
}

type Repo interface {
	UpsertVehicleStatus(ctx context.Context, vehicleID, plateNumber string, status map[string]interface{}) error
	GetVehicleStatus(ctx context.Context, vehicleID string) (map[string]interface{}, error)
	InsertTrip(ctx context.Context, trip model.Trip) error
	GetTripsSince(ctx context.Context, vehicleID string, since time.Time) ([]model.Trip, error)
	GetAllVehicleIDs(ctx context.Context) ([]string, error)
}

type RedisClient interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Ping(ctx context.Context) error
}
