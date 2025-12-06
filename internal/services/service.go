package services

import "sync"

type Customer struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Order struct {
	ID         int   `json:"id"`
	CustomerID int   `json:"customer_id"`
	AmountCents int64 `json:"amount_cents"`
}

var customers = []Customer{
	{ID: 1, Name: "Alice"},
	{ID: 2, Name: "Bob"},
}

var orders = map[int][]Order{
	1: {
		{ID: 11, CustomerID: 1, AmountCents: 100},
		{ID: 12, CustomerID: 1, AmountCents: 200},
	},
	2: {
		{ID: 21, CustomerID: 2, AmountCents: 150},
	},
}

var mu sync.Mutex

func GetCustomers() []Customer {
	mu.Lock()
	defer mu.Unlock()
	return customers
}

func GetOrdersForCustomer(id int) []Order {
	mu.Lock()
	defer mu.Unlock()
	if o, ok := orders[id]; ok {
		return o
	}
	return []Order{}
}
