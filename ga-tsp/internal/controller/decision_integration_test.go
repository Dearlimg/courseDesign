package controller

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"simple_tuan/internal/models"
)

func TestDecisionFlow(t *testing.T) {
	mux := NewMux()
	selection := `{"orders":[{"id":"A","name":"A","x":3,"y":0,"load":1,"income":100},{"id":"B","name":"B","x":0,"y":4,"load":1,"income":200},{"id":"C","name":"C","x":3,"y":4,"load":1,"income":300}],"capacity":3}`
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("POST", "/api/dispatch/select", strings.NewReader(selection)))
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var selected models.SelectionResult
	if err := json.Unmarshal(w.Body.Bytes(), &selected); err != nil {
		t.Fatal(err)
	}
	request := models.RouteRequest{Depot: models.Point{Name: "站"}, Points: []models.Point{}}
	for _, o := range selected.Selected {
		request.Points = append(request.Points, models.Point{Name: o.Name, X: o.X, Y: o.Y, OrderIDs: []string{o.ID}})
	}
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("POST", "/api/dispatch/route", strings.NewReader(string(body))))
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var route models.RouteResult
	if err := json.Unmarshal(w.Body.Bytes(), &route); err != nil {
		t.Fatal(err)
	}
	if len(route.Stops) != 4 || route.Distance != 14 {
		t.Fatalf("route %+v", route)
	}
}
