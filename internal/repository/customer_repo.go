package repository

import "go-backend-puzzle/internal/models"

var customers = []models.Customer{
	{ID: 1, Name: "Alice"},
	{ID: 2, Name: "Bob"},
}

func FetchAllCustomers() []models.Customer {
	return customers
}

func FetchCustomerByID(id int) *models.Customer {
	for _, c := range customers {
		if c.ID == id {
			return &c
		}
	}
	return nil
}
