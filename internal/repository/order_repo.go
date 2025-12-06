package repository

import "go-backend-puzzle/internal/models"

var orders = []models.Order{
	{ID: 1, CustomerID: 1, Amount: 100.5, Item: "Book"},
	{ID: 2, CustomerID: 1, Amount: 50, Item: "Pen"},
	{ID: 3, CustomerID: 2, Amount: 75, Item: "Notebook"},
}

func FetchOrdersForCustomers(customerIDs []int) map[int][]models.Order {
	result := make(map[int][]models.Order)
	for _, id := range customerIDs {
		for _, o := range orders {
			if o.CustomerID == id {
				result[id] = append(result[id], o)
			}
		}
	}
	return result
}

func FetchAllOrders() []models.Order {
	return orders
}
