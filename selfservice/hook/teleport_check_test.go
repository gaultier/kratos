package hook_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/gobuffalo/httptest"
	"github.com/ory/kratos/selfservice/flow/login"
	"github.com/ory/kratos/selfservice/hook"
	"github.com/ory/kratos/session"
	"github.com/ory/kratos/ui/node"
	"github.com/stretchr/testify/require"
)

func TestHaversineDistance(t *testing.T) {
	p1 := hook.DegPos(36.12, -86.67)
	p2 := hook.DegPos(33.94, -118.40)
	dist := hook.HaversineDistanceMeters(p1, p2)

	require.InEpsilon(t, 2887259.0, dist, 0.1)
}

func str(s string) *string {
	res := s
	return &res
}

func teleportCheckWithDevices(devices []session.Device) error {
	w := httptest.NewRecorder()
	var r http.Request
	f := &login.Flow{}
	var n node.UiNodeGroup

	h := hook.NewTeleportCheck()
	s := session.Session{
		Devices: devices,
	}
	err := h.ExecuteLoginPostHook(w, &r, n, f, &s)
	return err
}

func TestHookTeleportCheckLessThanTwoDevices(t *testing.T) {
	require.NoError(t, teleportCheckWithDevices([]session.Device{}))
	now := time.Now()
	require.NoError(t, teleportCheckWithDevices([]session.Device{
		{Location: str("Munich, Germany"), CreatedAt: now},
	}))
}

func TestHookTeleportCheckDistanceOk(t *testing.T) {
	now := time.Now()
	require.NoError(t, teleportCheckWithDevices([]session.Device{
		{Location: str("Munich, Germany"), CreatedAt: now},
		{Location: str("Paris, France"), CreatedAt: now.Add(5 * time.Hour)},
	}))
}

func TestHookTeleportCheckTooMuchDistance(t *testing.T) {
	now := time.Now()
	require.Error(t, teleportCheckWithDevices([]session.Device{
		{Location: str("Munich, Germany"), CreatedAt: now},
		{Location: str("Paris, France"), CreatedAt: now.Add(1 * time.Second)},
	}))
}
