package codexoauth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCallbackHandlerRejectsInvalidState(t *testing.T) {
	resultCh := make(chan callbackResult, 1)
	returnCh := make(chan struct{}, 1)
	handler := callbackHandler(callbackConfig{
		expectedState: "expected",
		returnToken:   "return-token",
	}, resultCh, returnCh)

	req := httptest.NewRequest(http.MethodGet, "/auth/callback?state=wrong&code=code", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest)
	}
	select {
	case result := <-resultCh:
		if result.err == nil {
			t.Fatal("expected state mismatch error")
		}
	case <-time.After(time.Second):
		t.Fatal("callback result was not reported")
	}
}

func TestCallbackHandlerReturnToken(t *testing.T) {
	resultCh := make(chan callbackResult, 1)
	returnCh := make(chan struct{}, 1)
	returned := false
	handler := callbackHandler(callbackConfig{
		expectedState: "expected",
		returnToken:   "return-token",
		onReturn: func() {
			returned = true
		},
	}, resultCh, returnCh)

	req := httptest.NewRequest(http.MethodGet, "/auth/return?token=return-token", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	if !returned {
		t.Fatal("return callback was not invoked")
	}
	select {
	case <-returnCh:
	case <-time.After(time.Second):
		t.Fatal("return signal was not sent")
	}
}
