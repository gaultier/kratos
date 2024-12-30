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

var (
	_                     login.PostHookExecutor = new(TeleportCheck)
	locationToGeoPosition                        = map[string]GeoPosition{
		"Munich, Germany":             DegPos(48.137154, 11.576124),
		"Paris, France":               DegPos(48.864716, 2.349014),
		"Taufkirchen, Germany":        DegPos(48.14927, 12.45652),
		"Taufkirchen (Vils), Germany": DegPos(48.34347, 12.13063),
		// TODO: more.
	}
)

type (
	GeoPosition struct {
		φ float64 // latitude, radians
		ψ float64 // longitude, radians
	}
	TeleportCheck struct {
	}
)

func NewTeleportCheck() *TeleportCheck {
	return &TeleportCheck{}
}

func haversine(θ float64) float64 {
	return .5 * (1 - math.Cos(θ))
}

func DegPos(lat, lon float64) GeoPosition {
	return GeoPosition{lat * math.Pi / 180, lon * math.Pi / 180}
}

// References:
// - https://en.wikipedia.org/wiki/Haversine_formula
// - https://rosettacode.org/wiki/Haversine_formula
func HaversineDistanceMeters(p1, p2 GeoPosition) float64 {
	rEarthMeters := 6372_800.0

	return 2 * rEarthMeters * math.Asin(math.Sqrt(haversine(p2.φ-p1.φ)+
		math.Cos(p1.φ)*math.Cos(p2.φ)*haversine(p2.ψ-p1.ψ)))
}

func (e *TeleportCheck) ExecuteLoginPostHook(_ http.ResponseWriter, r *http.Request, _ node.UiNodeGroup, _ *login.Flow, s *session.Session) error {
	return otelx.WithSpan(r.Context(), "selfservice.hook.TeleportCheck.ExecuteLoginPostHook", func(ctx context.Context) error {
		// If there is zero or one location, nothing to check.
		if len(s.Devices) <= 1 {
			return nil
		}

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

		// TODO: Move to configuration.
		maxAcceptableSpeedKilometerPerHour := float64(1000)
		maxAcceptableSpeedMeterPerSecond := maxAcceptableSpeedKilometerPerHour * 3.6

		if speedMeterPerSecond > maxAcceptableSpeedMeterPerSecond {
			return login.ErrTeleported
		}

		return nil
	})
}
