package health_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	api "github.com/Parcifallll/trip-go/internal/generated"
	"github.com/Parcifallll/trip-go/internal/http/health"
)

func decodeStatus(t *testing.T, rec *httptest.ResponseRecorder) api.HealthResponse {
	t.Helper()

	var resp api.HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	return resp
}

func TestHandler_Health(t *testing.T) {
	ctrl := gomock.NewController(t)
	handler := health.New(NewMockpinger(ctrl), time.Second)
	rec := httptest.NewRecorder()

	handler.Health(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if resp := decodeStatus(t, rec); resp.Status != api.Ok {
		t.Errorf("status = %q, want %q", resp.Status, api.Ok)
	}
}

func TestHandler_Ready(t *testing.T) {
	tests := []struct {
		name       string
		pingErr    error
		wantStatus int
		wantBody   string
	}{
		{name: "database is available", wantStatus: http.StatusOK, wantBody: string(api.Ok)},
		{
			name:       "database is unavailable",
			pingErr:    errors.New("connection refused"),
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   string(api.Unavailable),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			db := NewMockpinger(ctrl)
			db.EXPECT().Ping(gomock.Any()).Return(tc.pingErr)
			rec := httptest.NewRecorder()

			health.New(db, time.Second).Ready(rec, httptest.NewRequest(http.MethodGet, "/ready", nil))

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if resp := decodeStatus(t, rec); string(resp.Status) != tc.wantBody {
				t.Errorf("status = %q, want %q", resp.Status, tc.wantBody)
			}
		})
	}
}
