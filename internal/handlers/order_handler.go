package handlers

import (
    "net/http"
    "go-backend-puzzle/internal/services"
)

func OrderHandler(w http.ResponseWriter, r *http.Request) {
    orderService := services.NewOrderService()
    _ = orderService
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("orders ok"))
}
