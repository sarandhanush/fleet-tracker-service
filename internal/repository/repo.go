package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"fleet-tracker-service/internal/model"

	_ "github.com/lib/pq"
)

var (
	ErrNoTrips        = errors.New("trips not found")
	ErrNoVehicleFound = errors.New("Vehicle not found")
)

type Repo struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) UpsertVehicleStatus(ctx context.Context, vehicleID, plateNumber string, status map[string]interface{}) error {
	b, err := json.Marshal(status)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, `
    INSERT INTO vehicle (id, plate_number, last_status)
    VALUES ($1, $2, $3::jsonb)
    ON CONFLICT (id) DO UPDATE
      SET last_status = EXCLUDED.last_status
`, vehicleID, plateNumber, string(b))
	return err
}

func (r *Repo) GetVehicleStatus(ctx context.Context, vehicleID string) (map[string]interface{}, error) {
	var b []byte
	row := r.db.QueryRowContext(ctx, `SELECT last_status FROM vehicle WHERE id = $1`, vehicleID)
	if err := row.Scan(&b); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoVehicleFound
		}
		return nil, err
	}

	var out map[string]interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repo) InsertTrip(ctx context.Context, t model.Trip) error {
	_, err := r.db.ExecContext(ctx, `
        INSERT INTO trips (id, vehicle_id, start_time, end_time, mileage, avg_speed)
        VALUES ($1, $2, $3, $4, $5, $6)
    `, t.ID, t.VehicleID, t.StartTime, t.EndTime, t.Mileage, t.AvgSpeed)
	return err
}

func (r *Repo) GetTripsSince(ctx context.Context, vehicleID string, since time.Time) ([]model.Trip, error) {
	rows, err := r.db.QueryContext(ctx, `
        SELECT id, vehicle_id, start_time, end_time, mileage, avg_speed
        FROM trips
        WHERE vehicle_id = $1 AND start_time >= $2
        ORDER BY start_time DESC
    `, vehicleID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []model.Trip
	for rows.Next() {
		var t model.Trip
		if err := rows.Scan(&t.ID, &t.VehicleID, &t.StartTime, &t.EndTime, &t.Mileage, &t.AvgSpeed); err != nil {
			return nil, err
		}
		res = append(res, t)
	}

	if len(res) == 0 {
		return nil, ErrNoTrips
	}

	return res, nil
}

func (r *Repo) GetAllVehicleIDs(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id FROM vehicle")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vehicleIDs []string
	for rows.Next() {
		var vehicleID string
		if err := rows.Scan(&vehicleID); err != nil {
			return nil, err
		}
		vehicleIDs = append(vehicleIDs, vehicleID)
	}

	return vehicleIDs, nil
}
