package hook

import (
	"context"
	"math"
	"net/http"

	"github.com/ory/kratos/selfservice/flow/login"
	"github.com/ory/kratos/session"
	"github.com/ory/kratos/ui/node"
	"github.com/ory/x/otelx"
)

// The teleport check is a post login hook that verifies that the user did not 'teleport',
// i.e. did not attempt subsequent logins from two very far away locations in 'short' succession.
// 'Short' here is not thought in terms of time but instead, in terms of speed.
// If the user would have needed to go faster than a commercial plane to go from the previous location
// to the current one, we reject it.

var (
	_ login.PostHookExecutor = new(TeleportCheck)
	// We keep in-memory a compile time, read-only mapping of the location in a free flowing string format,
	// to a latitude and longitude (stored in radians but that is an implementation detail).
	locationToGeoPosition = map[string]GeoPosition{
		"Munich, Germany": DegPos(48.137154, 11.576124),
		"Paris, France":   DegPos(48.864716, 2.349014),
		// The string does not require a specific formatting:
		"Taufkirchen, Germany":        DegPos(48.14927, 12.45652),
		"Taufkirchen (Vils), Germany": DegPos(48.34347, 12.13063),
		// TODO: more.
	}
)

type (
	GeoPosition struct {
		// Optimization: we could use f32 or even f16.
		φ float64 // latitude, radians
		ψ float64 // longitude, radians
	}
	TeleportCheck struct {
	}
)

func NewTeleportCheck() *TeleportCheck {
	return &TeleportCheck{}
}

// Math references:
// - https://en.wikipedia.org/wiki/Haversine_formula
// - https://rosettacode.org/wiki/Haversine_formula
func haversine(θ float64) float64 {
	return .5 * (1 - math.Cos(θ))
}

func DegPos(lat, lon float64) GeoPosition {
	return GeoPosition{lat * math.Pi / 180, lon * math.Pi / 180}
}

func HaversineDistanceMeters(p1, p2 GeoPosition) float64 {
	rEarthMeters := 6372_800.0

	return 2 * rEarthMeters * math.Asin(math.Sqrt(haversine(p2.φ-p1.φ)+
		math.Cos(p1.φ)*math.Cos(p2.φ)*haversine(p2.ψ-p1.ψ)))
}

func (e *TeleportCheck) ExecuteLoginPostHook(_ http.ResponseWriter, r *http.Request, _ node.UiNodeGroup, _ *login.Flow, s *session.Session) error {
	return otelx.WithSpan(r.Context(), "selfservice.hook.TeleportCheck.ExecuteLoginPostHook", func(ctx context.Context) error {
		// If there is zero or one location, nothing we can check.
		if len(s.Devices) <= 1 {
			return nil
		}

		// We assume that the check has been run for all previous location.
		// Hence we only need to check the new (i.e. 'current') one with the previous one.
		locationPrevious := s.Devices[len(s.Devices)-2].Location
		locationCurrent := s.Devices[len(s.Devices)-1].Location

		// Fail open: if the location is absent/unknown, it is ok.
		geoPositionPrevious, ok := locationToGeoPosition[*locationPrevious]
		if !ok {
			return nil
		}

		geoPositionCurrent, ok := locationToGeoPosition[*locationCurrent]
		if !ok {
			return nil
		}

		distance := HaversineDistanceMeters(geoPositionPrevious, geoPositionCurrent)

		timePrevious := s.Devices[len(s.Devices)-2].CreatedAt
		timeCurrent := s.Devices[len(s.Devices)-1].CreatedAt
		duration := timeCurrent.Sub(timePrevious).Seconds()

		// Avoid divide by zero later.
		if duration == 0 {
			duration = math.SmallestNonzeroFloat64
		}

		speedMeterPerSecond := distance / duration

		// TODO: Move this constant to configuration?
		// That is the top speed of a commercial plane.
		maxAcceptableSpeedKilometerPerHour := float64(1000)
		maxAcceptableSpeedMeterPerSecond := maxAcceptableSpeedKilometerPerHour * 3.6

		if speedMeterPerSecond > maxAcceptableSpeedMeterPerSecond {
			return login.ErrTeleported
		}

		return nil
	})
}
