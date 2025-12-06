package models

type Customer struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Orders []Order `json:"orders"`
	TotalAmount float64 `json:"total_amount"`
}
