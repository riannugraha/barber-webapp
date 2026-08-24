package availability

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler_RegisterRoutes(t *testing.T) {
	fx := buildFixtures()
	svc := NewService(fx.repo)
	h := NewHandler(svc)
	e := echo.New()
	g := e.Group("/api/v1")
	h.RegisterRoutes(g)
	// Ensure route registered
	routes := e.Routes()
	found := false
	for _, r := range routes {
		if r.Method == http.MethodGet && r.Path == "/api/v1/availability/slots" {
			found = true
			break
		}
	}
	assert.True(t, found, "RegisterRoutes should mount GET /availability/slots")
}

func TestHandler_GetSlots_ServiceInvalidErrorMapsTo422(t *testing.T) {
	fx := buildFixtures()
	// Inject GetService to return invalid substring error to trigger handler's isInvalidError
	fx.repo.errGetService = assert.AnError
	// Make error message contain "invalid"
	fx.repo.errGetService = &customErr{"invalid service id from db"}
	svc := NewService(fx.repo)
	h := NewHandler(svc)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/availability/slots?serviceId="+fx.svcClassicID.String()+"&date=2025-11-10", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	_ = h.GetSlots(c)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

type customErr struct{ msg string }
func (e *customErr) Error() string { return e.msg }

func TestHandler_GetSlots_ServiceNotFoundMapsTo404_Already(t *testing.T) {
	// already covered but ensure
	fx := buildFixtures()
	svc := NewService(fx.repo)
	h := NewHandler(svc)
	e := echo.New()
	// use not found via fake uuid
	fake := "00000000-0000-0000-0000-000000000000"
	req := httptest.NewRequest(http.MethodGet, "/availability/slots?serviceId="+fake+"&date=2025-11-10", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	_ = h.GetSlots(c)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestIsNotFoundError_NilAndNegative(t *testing.T) {
	assert.False(t, isNotFoundError(nil))
	assert.False(t, isNotFoundError(&customErr{"other error"}))
	assert.True(t, isNotFoundError(&customErr{"entity not found"}))
}
