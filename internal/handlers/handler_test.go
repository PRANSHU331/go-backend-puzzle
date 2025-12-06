package handlers

import (
	"testing"
	"net/http/httptest"
)

func TestGetCustomers(t *testing.T) {
	req := httptest.NewRequest("GET", "/customers", nil)
	w := httptest.NewRecorder()
	GetCustomers(w, req)
	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestGetOrders(t *testing.T) {
	req := httptest.NewRequest("GET", "/orders", nil)
	w := httptest.NewRecorder()
	GetOrders(w, req)
	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}
