package hook

import (
	"context"
	"net/http"

	"github.com/ory/kratos/selfservice/flow/login"
	"github.com/ory/kratos/session"
	"github.com/ory/kratos/ui/node"
	"github.com/ory/x/otelx"
)

var _ login.PostHookExecutor = new(TeleportCheck)

type (
	TeleportCheck struct {
	}
)

func NewTeleportCheck() *TeleportCheck {
	return &TeleportCheck{}
}

func (e *TeleportCheck) ExecuteLoginPostHook(_ http.ResponseWriter, r *http.Request, _ node.UiNodeGroup, _ *login.Flow, s *session.Session) error {
	return otelx.WithSpan(r.Context(), "selfservice.hook.TeleportCheck.ExecuteLoginPostHook", func(ctx context.Context) error {
		// TODO
		return nil
	})
}
