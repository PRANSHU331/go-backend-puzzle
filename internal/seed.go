package main

import (
	"go-backend-puzzle/internal/models"
	"fmt"
)

func SeedData() {
	customers := []models.Customer{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
	}
	orders := []models.Order{
		{ID: 1, CustomerID: 1, Amount: 100.5, Item: "Book"},
		{ID: 2, CustomerID: 1, Amount: 50, Item: "Pen"},
		{ID: 3, CustomerID: 2, Amount: 75, Item: "Notebook"},
	}
	fmt.Println("Seeded", len(customers), "customers and", len(orders), "orders")
}

func main() {
	SeedData()
}
