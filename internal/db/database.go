package db

import "sync"

type Customer struct {
	ID    int
	Name  string
}

type Order struct {
	ID         int
	CustomerID int
	AmountCents int
}

var (
	customers []Customer
	orders    []Order
	once      sync.Once
)

func seed() {
	customers = []Customer{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
		{ID: 3, Name: "Charlie"},
	}

	orders = []Order{
		{ID: 1, CustomerID: 1, AmountCents: 1200},
		{ID: 2, CustomerID: 1, AmountCents: 800},
		{ID: 3, CustomerID: 2, AmountCents: 500},
	}
}

func initDB() {
	once.Do(seed)
}

func GetAllCustomers() ([]Customer, error) {
    initDB()
    return customers, nil
}


func GetAllOrders() ([]Order, error) {
    initDB()
    return orders, nil
}

func GetQueryCounter() int {
    return 1 
}