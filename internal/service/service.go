package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"fleet-tracker-service/internal/model"
	"fleet-tracker-service/internal/repository"

	"github.com/redis/go-redis/v9"
)

type IngestPayload struct {
	VehicleID   string                 `json:"vehicle_id"`
	PlateNumber string                 `json:"plate_number"`
	Status      map[string]interface{} `json:"status"`
}

type Service struct {
	repo *repository.Repo
	rdb  *redis.Client
}

func NewService(r *repository.Repo, rdb *redis.Client) *Service {
	return &Service{repo: r, rdb: rdb}
}

func cacheKeyStatus(vehicleID string) string {
	return fmt.Sprintf("vehicle:%s:status", vehicleID)
}

func randID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Service) Ingest(ctx context.Context, p IngestPayload) error {
	if p.VehicleID == "" || p.Status == nil {
		return errors.New("invalid payload")
	}

	if err := s.repo.UpsertVehicleStatus(ctx, p.VehicleID, p.PlateNumber, p.Status); err != nil {
		return err
	}

	vehicleUUID, err := uuid.Parse(p.VehicleID)
	if err != nil {
		return fmt.Errorf("invalid vehicle_id: %w", err)
	}

	if tsRaw, ok := p.Status["timestamp"].(string); ok {
		if ts, err := time.Parse(time.RFC3339, tsRaw); err == nil {
			t := model.Trip{
				ID:        uuid.New(),
				VehicleID: vehicleUUID,
				StartTime: ts.Add(-1 * time.Minute),
				EndTime:   ts,
				Mileage:   0.5,
				AvgSpeed:  30.0,
			}
			_ = s.repo.InsertTrip(ctx, t)
		}
	}
	key := cacheKeyStatus(p.VehicleID)
	b, _ := json.Marshal(p.Status)
	if err := s.rdb.Set(ctx, key, b, 5*time.Minute).Err(); err != nil {
		fmt.Println("redis set error:", err)
	}
	return nil
}

func (s *Service) GetStatus(ctx context.Context, vehicleID string) (map[string]interface{}, error) {
	key := cacheKeyStatus(vehicleID)
	vcmd := s.rdb.Get(ctx, key)
	var raw string
	if err := vcmd.Scan(&raw); err == nil {
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &m); err == nil {
			return m, nil
		}
	}
	st, err := s.repo.GetVehicleStatus(ctx, vehicleID)
	if err != nil {
		return nil, err
	}
	if st == nil {
		return nil, nil
	}
	if err := s.rdb.Set(ctx, key, st, 5*time.Minute).Err(); err != nil {
		fmt.Println("redis set error:", err)
	}
	return st, nil
}

func (s *Service) GetTripsLast24h(ctx context.Context, vehicleID string) ([]model.Trip, error) {
	since := time.Now().Add(-24 * time.Hour)
	return s.repo.GetTripsSince(ctx, vehicleID, since)
}

func (s *Service) StartSimulator(ctx context.Context) {
	go func() {
		vehicleIDs, err := s.repo.GetAllVehicleIDs(ctx)
		if err != nil {
			log.Printf("Error fetching vehicle IDs: %v", err)
			return
		}

		for {
			select {
			case <-ctx.Done():
				return
			default:
				for _, vid := range vehicleIDs {
					now := time.Now().UTC()
					status := map[string]interface{}{
						"location":  []float64{55.296249 + randOffset(), 25.276987 + randOffset()},
						"speed":     40 + randFloat(0, 30),
						"timestamp": now.Format(time.RFC3339),
					}

					_ = s.Ingest(ctx, IngestPayload{VehicleID: vid, Status: status})

					log.Printf("Simulated data for Vehicle ID: %s, Status: %v", vid, status)
				}
				time.Sleep(2 * time.Second)
			}
		}
	}()
}

// small helpers for randomness
func randOffset() float64 {
	return (randFloat(-0.001, 0.001))
}

func randFloat(min, max float64) float64 {
	return min + (max-min)*float64(time.Now().UnixNano()%1000)/1000.0
}
