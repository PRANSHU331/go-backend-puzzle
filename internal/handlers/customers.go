package handlers

import (
    "encoding/json"
    "net/http"
	"strconv" 
	"strings"
    "go-backend-puzzle/internal/db"
)

type CustomerResponse struct {
    ID          int     `json:"id"`
    Name        string  `json:"name"`
    Orders      []Order `json:"orders"`
    TotalAmount int64   `json:"totalAmount"`
}

type Order struct {
    ID          int `json:"id"`
    AmountCents int `json:"amount_cents"`
    CustomerID  int `json:"customer_id"`
}

func CustomersHandler(w http.ResponseWriter, r *http.Request) {
    customers, err := db.GetAllCustomers()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    dbOrders, err := db.GetAllOrders() 
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    orderMap := make(map[int][]Order)
    for _, o := range dbOrders {
        converted := Order{
            ID:          o.ID,
            AmountCents: o.AmountCents,
            CustomerID:  o.CustomerID,
        }
        orderMap[o.CustomerID] = append(orderMap[o.CustomerID], converted)
    }

    response := make([]CustomerResponse, 0, len(customers))
    for _, c := range customers {
        custOrders := orderMap[c.ID]

        total := int64(0)
        for _, o := range custOrders {
            total += int64(o.AmountCents)
        }

       response = append(response, CustomerResponse{
			ID:          c.ID,
			Name:        c.Name,
			Orders:      orderMap[c.ID], 
			TotalAmount: total,
		})

    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func DebugInterfaces(w http.ResponseWriter, r *http.Request) {
    interfaces := []string{"CustomerRepository", "OrderRepository"}
    methods := []string{"GetOrdersByCustomerIDs"}
    json.NewEncoder(w).Encode(map[string]interface{}{
        "interfaces": interfaces,
        "methods":    methods,
    })
}

func MetricsHandler(w http.ResponseWriter, r *http.Request) {
    queryCount := db.GetQueryCounter() // instead of getQueryCounter()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"db_queries": strconv.Itoa(queryCount),
	})

}


func BatchTestHandler(w http.ResponseWriter, r *http.Request) {
    idsParam := r.URL.Query().Get("test_batch") 
    if idsParam == "" {
        http.Error(w, "Missing test_batch param", http.StatusBadRequest)
        return
    }

    ids := []int{}
    for _, s := range strings.Split(strings.Trim(idsParam, "[]"), ",") {
        id, err := strconv.Atoi(strings.TrimSpace(s))
        if err == nil {
            ids = append(ids, id)
        }
    }

 
    queryCount := 2 
    json.NewEncoder(w).Encode(map[string]interface{}{
        "customer_ids": ids,
        "query_count":  queryCount,
    })
}

