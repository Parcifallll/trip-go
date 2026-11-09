package create

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/Parcifallll/trip-go/internal/domain/trip"
	api "github.com/Parcifallll/trip-go/internal/generated"
	"github.com/Parcifallll/trip-go/internal/model"
)

const maxBodyBytes = 1 << 20

var (
	requiredFields = []string{"user_id", "driver_id", "start_point", "end_point", "price"}
	pointFields    = []string{"start_point", "end_point"}
	coordFields    = []string{"latitude", "longitude"}
)

func parseRequest(w http.ResponseWriter, r *http.Request) (model.CreateTripInput, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return model.CreateTripInput{}, invalid("body", "cannot be read")
	}

	if err := checkRequired(raw); err != nil {
		return model.CreateTripInput{}, err
	}

	var body api.CreateTripJSONRequestBody

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		return model.CreateTripInput{}, invalid("body", "does not match the schema")
	}

	return model.CreateTripInput{
		UserID:     body.UserId,
		DriverID:   body.DriverId,
		StartPoint: trip.Coordinates{Latitude: body.StartPoint.Latitude, Longitude: body.StartPoint.Longitude},
		EndPoint:   trip.Coordinates{Latitude: body.EndPoint.Latitude, Longitude: body.EndPoint.Longitude},
		Price:      body.Price,
	}, nil
}

func checkRequired(raw []byte) error {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		return invalid("body", "must be a valid JSON object")
	}

	for _, field := range requiredFields {
		if !present(top[field]) {
			return invalid(field, "is required")
		}
	}

	for _, field := range pointFields {
		var point map[string]json.RawMessage
		if err := json.Unmarshal(top[field], &point); err != nil {
			return invalid(field, "must be an object")
		}
		for _, coord := range coordFields {
			if !present(point[coord]) {
				return invalid(field+"."+coord, "is required")
			}
		}
	}

	return nil
}

func present(v json.RawMessage) bool {
	return len(v) > 0 && !bytes.Equal(v, []byte("null"))
}

func invalid(field, reason string) error {
	return &trip.ValidationError{Field: field, Reason: reason}
}
