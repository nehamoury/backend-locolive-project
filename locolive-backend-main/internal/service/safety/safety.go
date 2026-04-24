package safety

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// MaxSpeedKmH is 1000 km/h (approx jet speed). Anything faster is definitely fake.
	MaxSpeedKmH = 1000.0
	// Key prefix for last location
	lastLocationKeyPrefix = "safety:last_loc:"
)

type ValidationResult struct {
	Allowed   bool
	Reason    string
	ShouldBan bool
}

// Monitor handles safety checks like Fake GPS
type Monitor struct {
	redis *redis.Client
}

func NewMonitor(rdb *redis.Client) *Monitor {
	return &Monitor{redis: rdb}
}

// ValidateUserMovement checks if the user moved typically fast and analyzes patterns
func (s *Monitor) ValidateUserMovement(ctx context.Context, userID string, newLat, newLng float64) ValidationResult {
	key := lastLocationKeyPrefix + userID

	// Get last location and history
	res, err := s.redis.HGetAll(ctx, key).Result()
	if err != nil || len(res) == 0 {
		s.saveLastLocation(ctx, key, newLat, newLng, nil)
		return ValidationResult{Allowed: true}
	}

	lastLat := parseFloat(res["lat"])
	lastLng := parseFloat(res["lng"])
	lastTime, _ := time.Parse(time.RFC3339, res["time"])
	historyJSON := res["history"]

	now := time.Now()
	timeDiffHours := now.Sub(lastTime).Hours()

	if timeDiffHours <= 0 {
		return ValidationResult{Allowed: true}
	}

	// 1. Basic Velocity Check (Existing)
	distKm := haversineKm(lastLat, lastLng, newLat, newLng)
	speed := distKm / timeDiffHours

	if speed > MaxSpeedKmH {
		return ValidationResult{
			Allowed:   false,
			Reason:    "Impossible speed detected (" + formatFloat(speed) + " km/h)",
			ShouldBan: true,
		}
	}

	// 2. Pattern Analysis: Jitter & Grid Detection
	var history []struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	}
	_ = json.Unmarshal([]byte(historyJSON), &history)

	// Keep last 5 points
	history = append(history, struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	}{Lat: newLat, Lng: newLng})
	if len(history) > 5 {
		history = history[1:]
	}

	if len(history) >= 4 {
		// Detect "Zero Jitter" - Scripts often use exact decimals
		// Detect "Perfect Grid" - Straight lines only
		isPerfectGrid := true
		for i := 1; i < len(history); i++ {
			dLat := math.Abs(history[i].Lat - history[i-1].Lat)
			dLng := math.Abs(history[i].Lng - history[i-1].Lng)
			
			// If moving diagonally, it's less likely to be a simple grid bot
			if dLat > 0.000001 && dLng > 0.000001 {
				isPerfectGrid = false
				break
			}
		}

		if isPerfectGrid && distKm > 0.1 { // Only flag if actually moving
			return ValidationResult{
				Allowed:   false,
				Reason:    "Suspicious movement pattern (Grid-like)",
				ShouldBan: false, // Don't ban yet, just block this ping
			}
		}
	}

	// Valid, update last location and history
	histBytes, _ := json.Marshal(history)
	s.saveLastLocation(ctx, key, newLat, newLng, histBytes)
	return ValidationResult{Allowed: true}
}

func (s *Monitor) saveLastLocation(ctx context.Context, key string, lat, lng float64, history []byte) {
	data := map[string]interface{}{
		"lat":  lat,
		"lng":  lng,
		"time": time.Now().Format(time.RFC3339),
	}
	if history != nil {
		data["history"] = string(history)
	}
	s.redis.HSet(ctx, key, data)
	s.redis.Expire(ctx, key, 24*time.Hour)
}

// -- Helpers --

func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Radius of the earth in km
	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLon := (lon2 - lon1) * (math.Pi / 180.0)
	lat1 = lat1 * (math.Pi / 180.0)
	lat2 = lat2 * (math.Pi / 180.0)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1)*math.Cos(lat2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func formatFloat(f float64) string {
	return fmt.Sprintf("%.2f", f)
}
