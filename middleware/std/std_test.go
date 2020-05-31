package std_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"

	mmetrics "github.com/slok/go-http-metrics/internal/mocks/metrics"
	"github.com/slok/go-http-metrics/metrics"
	"github.com/slok/go-http-metrics/middleware"
	stdmiddleware "github.com/slok/go-http-metrics/middleware/std"
)

func getFakeHandler(statusCode int, responseBody string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(responseBody))
	})
}

func TestMiddleware(t *testing.T) {
	tests := map[string]struct {
		handlerID     string
		body          string
		statusCode    int
		req           *http.Request
		config        middleware.Config
		expHandlerID  string
		expService    string
		expMethod     string
		expSize       int64
		expStatusCode string
	}{
		"A default HTTP middleware should call the recorder to measure.": {
			statusCode:    http.StatusAccepted,
			body:          "Я бэтмен",
			req:           httptest.NewRequest(http.MethodGet, "/test", nil),
			expHandlerID:  "/test",
			expService:    "",
			expSize:       15,
			expMethod:     http.MethodGet,
			expStatusCode: "202",
		},

		"Using custom ID in the middleware should call the recorder to measure with that ID.": {
			handlerID:     "customID",
			body:          "I'm Batman",
			statusCode:    http.StatusTeapot,
			req:           httptest.NewRequest(http.MethodPost, "/test", nil),
			expHandlerID:  "customID",
			expService:    "",
			expSize:       10,
			expMethod:     http.MethodPost,
			expStatusCode: "418",
		},

		"Using grouped status code should group the status code.": {
			config:        middleware.Config{GroupedStatus: true},
			statusCode:    http.StatusGatewayTimeout,
			req:           httptest.NewRequest(http.MethodPatch, "/test", nil),
			expHandlerID:  "/test",
			expService:    "",
			expMethod:     http.MethodPatch,
			expStatusCode: "5xx",
		},

		"Using the service middleware option should set the service on the metrics.": {
			config:        middleware.Config{Service: "Yoda"},
			statusCode:    http.StatusContinue,
			body:          "May the force be with you",
			req:           httptest.NewRequest(http.MethodGet, "/test", nil),
			expHandlerID:  "/test",
			expService:    "Yoda",
			expSize:       25,
			expMethod:     http.MethodGet,
			expStatusCode: "100",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// Mocks.
			mr := &mmetrics.Recorder{}
			expHTTPReqProps := metrics.HTTPReqProperties{
				ID:      test.expHandlerID,
				Service: test.expService,
				Method:  test.expMethod,
				Code:    test.expStatusCode,
			}
			expHTTPProps := metrics.HTTPProperties{
				ID:      test.expHandlerID,
				Service: test.expService,
			}
			mr.On("ObserveHTTPRequestDuration", mock.Anything, expHTTPReqProps, mock.Anything).Once()
			mr.On("ObserveHTTPResponseSize", mock.Anything, expHTTPReqProps, test.expSize).Once()
			mr.On("AddInflightRequests", mock.Anything, expHTTPProps, 1).Once()
			mr.On("AddInflightRequests", mock.Anything, expHTTPProps, -1).Once()

			// Make the request.
			test.config.Recorder = mr
			m := middleware.New(test.config)
			h := stdmiddleware.Handler(test.handlerID, m, getFakeHandler(test.statusCode, test.body))
			h.ServeHTTP(httptest.NewRecorder(), test.req)

			mr.AssertExpectations(t)
		})
	}
}
