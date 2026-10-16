package httpapi

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/calm195/go-template/internal/user"
)

type fakeRepository struct{ err error }

func (r fakeRepository) Create(_ context.Context, u *user.User) error {
	u.ID = 1
	return r.err
}

func TestRoutes(t *testing.T) {
	for _, tc := range []struct {
		name, method, path, body string
		users                    *user.Service
		readyErr                 error
		status                   int
	}{
		{name: "live", method: "GET", path: "/healthz", status: 200},
		{name: "ready failure", method: "GET", path: "/readyz", readyErr: errors.New("offline"), status: 503},
		{name: "disabled database", method: "POST", path: "/api/v1/users", body: `{"name":"Ada"}`, status: 503},
		{name: "invalid input", method: "POST", path: "/api/v1/users", body: `{"name":" "}`, users: &user.Service{Repository: fakeRepository{}}, status: 400},
		{name: "created", method: "POST", path: "/api/v1/users", body: `{"name":"Ada"}`, users: &user.Service{Repository: fakeRepository{}}, status: 201},
		{name: "storage failure", method: "POST", path: "/api/v1/users", body: `{"name":"Ada"}`, users: &user.Service{Repository: fakeRepository{err: errors.New("secret DSN")}}, status: 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			core, logs := observer.New(zap.InfoLevel)
			router := New(zap.New(core), func(context.Context) error { return tc.readyErr }, tc.users)
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)
			if response.Code != tc.status {
				t.Fatalf("status %d, body %s", response.Code, response.Body.String())
			}
			if response.Header().Get("X-Request-ID") == "" {
				t.Fatal("missing request ID")
			}
			// 完成日志必须关联响应，并反映操作的实际结果。
			completed := logs.FilterMessage("http request completed").All()
			if len(completed) != 1 {
				t.Fatalf("expected one completion log, got %d", len(completed))
			}
			fields := completed[0].ContextMap()
			if fields["request_id"] != response.Header().Get("X-Request-ID") || fields["status"] != int64(tc.status) {
				t.Fatalf("incorrect completion fields: %v", fields)
			}
			created := logs.FilterMessage("create user completed").Len()
			if (tc.status == 201 && created != 1) || (tc.status != 201 && created != 0) {
				t.Fatalf("incorrect user completion count: %d", created)
			}
			for _, entry := range logs.All() {
				for _, field := range entry.Context {
					if strings.Contains(field.String, "secret") {
						t.Fatal("log leaked storage details")
					}
				}
			}
			if strings.Contains(response.Body.String(), "secret") {
				t.Fatal("error leaked storage details")
			}
		})
	}
}

func TestRecoveryKeepsCompletionLog(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	router := New(zap.New(core), func(context.Context) error {
		panic("secret payload")
	}, nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/readyz", nil))
	if response.Code != 500 {
		t.Fatalf("expected 500, got %d", response.Code)
	}
	// 必须先完成异常恢复，外层中间件才能记录最终的 HTTP 状态。
	completed := logs.FilterMessage("http request completed").All()
	if len(completed) != 1 || completed[0].ContextMap()["status"] != int64(500) {
		t.Fatalf("missing or incorrect completion log: %v", completed)
	}
	if logs.FilterMessage("request panic").Len() != 1 {
		t.Fatal("missing panic log")
	}
	for _, entry := range logs.All() {
		for _, field := range entry.Context {
			if strings.Contains(field.String, "secret") {
				t.Fatal("panic value leaked to logs")
			}
		}
	}
	if strings.Contains(response.Body.String(), "secret") {
		t.Fatal("panic value leaked to response")
	}
}
