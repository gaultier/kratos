package hook

import (
	"context"
	"net/http"

	"github.com/ory/kratos/selfservice/flow/login"
	"github.com/ory/kratos/session"
	"github.com/ory/kratos/ui/node"
	"github.com/ory/x/otelx"
)

var (
	_                login.PostHookExecutor = new(TeleportCheck)
	locationToLatLng                        = map[string]LatLng{
		"Munich, Germany":             {48.137154, 11.576124},
		"Paris, France":               {48.864716, 2.349014},
		"Taufkirchen, Germany":        {48.14927, 12.45652},
		"Taufkirchen (Vils), Germany": {48.34347, 12.13063},
		// TODO: more.
	}
)

type (
	LatLng struct {
		Latitude  float64
		Longitude float64
	}
	TeleportCheck struct {
	}
)

func NewTeleportCheck() *TeleportCheck {
	return &TeleportCheck{}
}

func haversineDistance(a, b LatLng) float64 {
	// TODO
	return 0.0
}

func (e *TeleportCheck) ExecuteLoginPostHook(_ http.ResponseWriter, r *http.Request, _ node.UiNodeGroup, _ *login.Flow, s *session.Session) error {
	return otelx.WithSpan(r.Context(), "selfservice.hook.TeleportCheck.ExecuteLoginPostHook", func(ctx context.Context) error {
		// If there is zero or one location, nothing to check.
		if len(s.Devices) <= 1 {
			return nil
		}

		locationPrevious := s.Devices[len(s.Devices)-2].Location
		locationCurrent := s.Devices[len(s.Devices)-1].Location

		latLngPrevious, ok := locationToLatLng[*locationPrevious]
		if !ok {
			return nil
		}

		latLngCurrent, ok := locationToLatLng[*locationCurrent]
		if !ok {
			return nil
		}

		distance := haversineDistance(latLngPrevious, latLngCurrent)

		timePrevious := s.Devices[len(s.Devices)-2].CreatedAt
		timeCurrent := s.Devices[len(s.Devices)-1].CreatedAt
		duration := timeCurrent.Sub(timePrevious).Seconds()

		// TODO: Avoid divide by zero.
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
