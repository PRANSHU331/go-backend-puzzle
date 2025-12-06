package main

import (
	"go-backend-puzzle/internal/handlers"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/health", handlers.HealthHandler)
	http.HandleFunc("/customers/", handlers.OrdersHandler) 
	http.HandleFunc("/customers", handlers.UsersHandler)
    http.HandleFunc("/customers?debug=interfaces", handlers.DebugInterfaces)
    http.HandleFunc("/customers?metrics=true", handlers.MetricsHandler)
    http.HandleFunc("/customers/batch", handlers.BatchTestHandler)

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
