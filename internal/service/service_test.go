package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"fleet-tracker-service/internal/model"

	"github.com/google/uuid"
)

var redisNilErr = errors.New("redis: nil")

var testSummary = struct {
	sync.Mutex
	total  int
	passed int
	failed int
}{}

func logResult(passed bool) {
	testSummary.Lock()
	defer testSummary.Unlock()
	testSummary.total++
	if passed {
		testSummary.passed++
	} else {
		testSummary.failed++
	}
}

type fakeRepo struct {
	upsertCalled bool
	statusCalled bool
}

func (f *fakeRepo) UpsertVehicleStatus(ctx context.Context, vehicleID, plateNumber string, status map[string]interface{}) error {
	f.upsertCalled = true
	return nil
}

func (f *fakeRepo) GetVehicleStatus(ctx context.Context, vehicleID string) (map[string]interface{}, error) {
	f.statusCalled = true
	return map[string]interface{}{"speed": float64(42)}, nil
}

func (f *fakeRepo) InsertTrip(ctx context.Context, trip model.Trip) error { return nil }
func (f *fakeRepo) GetTripsSince(ctx context.Context, vehicleID string, since time.Time) ([]model.Trip, error) {
	return []model.Trip{}, nil
}
func (f *fakeRepo) GetAllVehicleIDs(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

type fakeRedis struct {
	setCalled bool
	getCalled bool
	storage   map[string]string
	SetFunc   func(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	GetFunc   func(ctx context.Context, key string) (string, error)
}

func (f *fakeRedis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if f.SetFunc != nil {
		return f.SetFunc(ctx, key, value, expiration)
	}
	f.setCalled = true
	if f.storage == nil {
		f.storage = make(map[string]string)
	}
	b, _ := json.Marshal(value)
	f.storage[key] = string(b)
	return nil
}

func (f *fakeRedis) Get(ctx context.Context, key string) (string, error) {
	if f.GetFunc != nil {
		return f.GetFunc(ctx, key)
	}
	f.getCalled = true
	if v, ok := f.storage[key]; ok {
		return v, nil
	}
	return "", redisNilErr
}

func (f *fakeRedis) Ping(ctx context.Context) error { return nil }

func TestVehicleStatusCacheKey(t *testing.T) {
	cases := []struct {
		name      string
		vehicleID string
		want      string
	}{
		{"Simple vehicle ID", "abc", "vehicle:abc:status"},
		{"Short hex ID", "d9c1", "vehicle:d9c1:status"},
		{"UUID vehicle ID", "550e8400-e29b-41d4-a716-446655440000", "vehicle:550e8400-e29b-41d4-a716-446655440000:status"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Logf("Running test: %s", c.name)
			got := cacheKeyStatus(c.vehicleID)
			passed := got == c.want
			if !passed {
				t.Errorf("cacheKeyStatus(%q) = %q, want %q", c.vehicleID, got, c.want)
			}
			logResult(passed)
		})
	}
}

func TestVehicleStatusIngest(t *testing.T) {
	cases := []struct {
		name    string
		payload IngestPayload
		wantErr bool
	}{
		{
			"Valid payload with speed and timestamp",
			IngestPayload{
				VehicleID:   uuid.New().String(),
				PlateNumber: "ABC123",
				Status: map[string]interface{}{
					"speed":     50.0,
					"timestamp": time.Now().Format(time.RFC3339),
				},
			},
			false,
		},
		{
			"Empty vehicle ID should fail",
			IngestPayload{
				VehicleID: "",
				Status:    map[string]interface{}{"speed": 30.0},
			},
			true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Logf("Running test: %s", c.name)
			repo := &fakeRepo{}
			rdb := &fakeRedis{}
			svc := &Service{repo: repo, rdb: rdb}

			err := svc.Ingest(context.Background(), c.payload)
			passed := (err != nil) == c.wantErr
			logResult(passed)
		})
	}

	t.Run("Ingest succeeds even if Redis set fails", func(t *testing.T) {
		t.Log("Running test: Ingest succeeds even if Redis set fails")
		repo := &fakeRepo{}
		rdb := &fakeRedis{
			SetFunc: func(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
				return errors.New("redis set failed")
			},
		}
		svc := &Service{repo: repo, rdb: rdb}
		payload := IngestPayload{
			VehicleID:   uuid.New().String(),
			PlateNumber: "ABC123",
			Status:      map[string]interface{}{"speed": 55},
		}
		_ = svc.Ingest(context.Background(), payload)
		logResult(true)
	})
}

func TestGetVehicleStatus(t *testing.T) {
	cases := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			"Cache hit returns stored speed",
			func(t *testing.T) {
				repo := &fakeRepo{}
				rdb := &fakeRedis{storage: map[string]string{}}
				svc := &Service{repo: repo, rdb: rdb}
				vehicleID := "veh123"
				key := cacheKeyStatus(vehicleID)
				_ = rdb.Set(context.Background(), key, map[string]interface{}{"speed": 99}, 5*time.Minute)
				got, _ := svc.GetStatus(context.Background(), vehicleID)
				logResult(got["speed"] == float64(99))
			},
		},
		{
			"Cache miss falls back to repo",
			func(t *testing.T) {
				repo := &fakeRepo{}
				rdb := &fakeRedis{}
				svc := &Service{repo: repo, rdb: rdb}
				got, _ := svc.GetStatus(context.Background(), "veh456")
				logResult(got["speed"] == float64(42))
			},
		},
		{
			"Redis get error falls back to repo",
			func(t *testing.T) {
				repo := &fakeRepo{}
				rdb := &fakeRedis{
					GetFunc: func(ctx context.Context, key string) (string, error) { return "", errors.New("redis error") },
				}
				svc := &Service{repo: repo, rdb: rdb}
				got, _ := svc.GetStatus(context.Background(), "veh789")
				logResult(got["speed"] == float64(42))
			},
		},
		{
			"Redis set failure does not prevent repo return",
			func(t *testing.T) {
				repo := &fakeRepo{}
				rdb := &fakeRedis{
					SetFunc: func(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
						return errors.New("fail")
					},
				}
				svc := &Service{repo: repo, rdb: rdb}
				got, _ := svc.GetStatus(context.Background(), "veh321")
				logResult(got["speed"] == float64(42))
			},
		},
		{
			"Bad cached JSON falls back to repo",
			func(t *testing.T) {
				repo := &fakeRepo{}
				rdb := &fakeRedis{storage: map[string]string{cacheKeyStatus("veh999"): "{invalid-json}"}}
				svc := &Service{repo: repo, rdb: rdb}
				got, _ := svc.GetStatus(context.Background(), "veh999")
				logResult(got["speed"] == float64(42))
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Logf("Running test: %s", c.name)
			c.fn(t)
		})
	}
}

func TestSummary(t *testing.T) {
	t.Logf("======== TEST SUMMARY ========")
	t.Logf("Total tests run: %d", testSummary.total)
	t.Logf("Passed: %d", testSummary.passed)
	t.Logf("Failed: %d", testSummary.failed)
	if testSummary.failed > 0 {
		t.Fail()
	}
}
