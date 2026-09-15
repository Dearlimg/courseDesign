package controller

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDispatchSelectionAPI(t *testing.T) {
	for _, tc := range []struct {
		body   string
		status int
	}{
		{body: `{"orders":[{"id":"A","name":"点","x":1,"y":2,"load":1,"income":101}],"capacity":1}`, status: 200},
		{body: `{"orders":[],"capacity":1}`, status: 200},
		{body: `{"orders":[],"capacity":0}`, status: 400},
		{body: `{"orders":[],"capacity":1,"unknown":true}`, status: 400},
		{body: `{"orders":[],"capacity":1}{}`, status: 400},
	} {
		w := httptest.NewRecorder()
		NewMux().ServeHTTP(w, httptest.NewRequest("POST", "/api/dispatch/select", strings.NewReader(tc.body)))
		if w.Code != tc.status {
			t.Fatalf("%s: %d %s", tc.body, w.Code, w.Body.String())
		}
	}
}
