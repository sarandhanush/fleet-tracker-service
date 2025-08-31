package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type ServiceInterface interface {
	Ingest(ctx context.Context, p IngestPayload) error
	GetStatus(ctx context.Context, vehicleID string) (map[string]interface{}, error)
}

type MockService struct {
	mock.Mock
}

func (m *MockService) Ingest(ctx context.Context, p IngestPayload) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

func (m *MockService) GetStatus(ctx context.Context, vehicleID string) (map[string]interface{}, error) {
	args := m.Called(ctx, vehicleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func NewIngestHandler(svc ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload IngestPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		if err := svc.Ingest(r.Context(), payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "Ingest successful"}`))
	}
}

func NewGetStatusHandler(svc ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vehicleID := r.URL.Query().Get("vehicle_id")
		if vehicleID == "" {
			http.Error(w, "vehicle_id query parameter is required", http.StatusBadRequest)
			return
		}

		status, err := svc.GetStatus(r.Context(), vehicleID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if status == nil {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(status)
	}
}

func TestAPIHandlers(t *testing.T) {
	testService := new(MockService)

	router := http.NewServeMux()
	router.HandleFunc("/ingest", NewIngestHandler(testService))
	router.HandleFunc("/status", NewGetStatusHandler(testService))

	t.Run("Ingest with valid payload", func(t *testing.T) {
		ingestPayload := IngestPayload{
			VehicleID:   uuid.New().String(),
			PlateNumber: "XYZ-789",
			Status:      map[string]interface{}{"speed": 60.0},
		}
		jsonBody, _ := json.Marshal(ingestPayload)

		testService.On("Ingest", mock.Anything, ingestPayload).Return(nil).Once()

		request := httptest.NewRequest("POST", "/ingest", bytes.NewBuffer(jsonBody))
		request.Header.Set("Content-Type", "application/json")

		responseRecorder := httptest.NewRecorder()

		router.ServeHTTP(responseRecorder, request)

		assert.Equal(t, http.StatusOK, responseRecorder.Code)
		assert.Contains(t, responseRecorder.Body.String(), "Ingest successful")

		testService.AssertExpectations(t)
	})

	t.Run("Ingest with invalid payload", func(t *testing.T) {
		invalidBody := `{"vehicle_id": "123"`
		request := httptest.NewRequest("POST", "/ingest", bytes.NewBufferString(invalidBody))
		request.Header.Set("Content-Type", "application/json")

		responseRecorder := httptest.NewRecorder()

		router.ServeHTTP(responseRecorder, request)

		assert.Equal(t, http.StatusBadRequest, responseRecorder.Code)
		assert.Contains(t, responseRecorder.Body.String(), "Invalid request payload")

		testService.AssertNotCalled(t, "Ingest")
	})

	t.Run("GetStatus for existing vehicle", func(t *testing.T) {
		vehicleID := uuid.New().String()
		expectedStatus := map[string]interface{}{
			"speed":     45.5,
			"timestamp": time.Now().Format(time.RFC3339),
		}

		testService.On("GetStatus", mock.Anything, vehicleID).Return(expectedStatus, nil).Once()

		request := httptest.NewRequest("GET", "/status?vehicle_id="+vehicleID, nil)
		responseRecorder := httptest.NewRecorder()

		router.ServeHTTP(responseRecorder, request)

		assert.Equal(t, http.StatusOK, responseRecorder.Code)
		var actualStatus map[string]interface{}
		_ = json.NewDecoder(io.Reader(responseRecorder.Body)).Decode(&actualStatus)
		assert.Equal(t, expectedStatus["speed"], actualStatus["speed"])

		testService.AssertExpectations(t)
	})

	t.Run("GetStatus for non-existent vehicle", func(t *testing.T) {
		vehicleID := uuid.New().String()

		testService.On("GetStatus", mock.Anything, vehicleID).Return(nil, nil).Once()

		request := httptest.NewRequest("GET", "/status?vehicle_id="+vehicleID, nil)
		responseRecorder := httptest.NewRecorder()

		router.ServeHTTP(responseRecorder, request)

		assert.Equal(t, http.StatusNotFound, responseRecorder.Code)
		assert.Contains(t, responseRecorder.Body.String(), "Not Found")

		testService.AssertExpectations(t)
	})
}

func TestCacheKeyStatus(t *testing.T) {
	testCases := []struct {
		id   string
		want string
	}{
		{"abc", "vehicle:abc:status"},
		{"d9c1", "vehicle:d9c1:status"},
	}
	for _, tc := range testCases {
		got := cacheKeyStatus(tc.id)
		if got != tc.want {
			t.Errorf("cacheKeyStatus(%s) got %s, want %s", tc.id, got, tc.want)
		}
	}
}

func TestRandIDUnique(t *testing.T) {
	firstID := randID()
	secondID := randID()

	if firstID == secondID {
		t.Errorf("expected unique random IDs, but got duplicates: %s", firstID)
	}
}

func TestRedisSetGetIntegration(t *testing.T) {
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	ctx := context.Background()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skip("skipping Redis integration test: Redis not available at localhost:6379")
	}

	testKey := "test:fleet:key"
	testValue := `{"ok":true}`

	if err := redisClient.Set(ctx, testKey, testValue, 1*time.Minute).Err(); err != nil {
		t.Fatalf("failed to set value in Redis: %v", err)
	}

	var storedValue string
	if err := redisClient.Get(ctx, testKey).Scan(&storedValue); err != nil {
		t.Fatalf("failed to get value from Redis: %v", err)
	}

	assert.Equal(t, testValue, storedValue)
}
