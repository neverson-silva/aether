package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	appsdomain "aether/internal/modules/apps/domain"
	authhttp "aether/internal/modules/auth/http"
	servicesdomain "aether/internal/modules/services/domain"
)

type lifecycleAppReader struct {
	app *appsdomain.App
}

func (r lifecycleAppReader) GetApp(_ context.Context, _, _ uuid.UUID) (*appsdomain.App, error) {
	return r.app, nil
}

type lifecycleRequester struct {
	operation *servicesdomain.LifecycleOperation
	requested servicesdomain.LifecycleAction
}

func (r *lifecycleRequester) RequestBySpec(_ context.Context, specID, orgID uuid.UUID, kind servicesdomain.Kind, action servicesdomain.LifecycleAction) (*servicesdomain.LifecycleOperation, error) {
	r.requested = action
	return r.operation, nil
}

func TestAppLifecycleEndpointsAcceptWithoutRuntimeExecutor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	orgID := uuid.New()
	appID := uuid.New()
	serviceID := uuid.New()
	for _, test := range []struct {
		name    string
		action  servicesdomain.LifecycleAction
		status  servicesdomain.Status
		handler func(*Handler, *gin.Context)
	}{
		{name: "start", action: servicesdomain.LifecycleStart, status: servicesdomain.StatusStarting, handler: (*Handler).AppStart},
		{name: "stop", action: servicesdomain.LifecycleStop, status: servicesdomain.StatusStopping, handler: (*Handler).AppStop},
	} {
		t.Run(test.name, func(t *testing.T) {
			requester := &lifecycleRequester{operation: &servicesdomain.LifecycleOperation{ID: uuid.New(), ServiceID: serviceID, OrgID: orgID, SpecID: appID, Kind: servicesdomain.KindApp, Action: test.action, Status: "accepted"}}
			apps := lifecycleAppReader{app: &appsdomain.App{ID: appID, OrgID: orgID}}
			handler := New(nil, apps, nil, "", nil).WithLifecycle(requester)
			request := httptest.NewRequest(http.MethodPost, "/apps/"+appID.String()+"/"+test.name, nil)
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)
			context.Request = request
			context.Params = gin.Params{{Key: "appID", Value: appID.String()}}
			context.Set(authhttp.ContextOrgID, orgID)
			test.handler(handler, context)
			if recorder.Code != http.StatusAccepted {
				t.Fatalf("expected HTTP %d, got %d: %s", http.StatusAccepted, recorder.Code, recorder.Body.String())
			}
			if requester.requested != test.action {
				t.Fatalf("expected %s command, got %s", test.action, requester.requested)
			}
			if !strings.Contains(recorder.Body.String(), `"status":"accepted"`) || !strings.Contains(recorder.Body.String(), `"state":"`+string(test.status)+`"`) {
				t.Fatalf("expected accepted command and state %q, got %s", test.status, recorder.Body.String())
			}
		})
	}
}
