package models

type Order struct {
	ID         int     `json:"id"`
	CustomerID int     `json:"customer_id"`
	Amount     float64 `json:"amount"`
	Item       string  `json:"item"`
}
