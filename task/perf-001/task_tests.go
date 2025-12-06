package perf001_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)


type Customer struct {
	ID   int
	Name string
}

type Order struct {
	ID         int
	CustomerID int
	AmountCents int64
}

type CustomerWithOrders struct {
	ID          int
	Name        string
	Orders      []Order
	TotalAmount int64
}

type CustomerRepo interface {
	ListCustomers(ctx context.Context) ([]Customer, error)
}

type OrderRepo interface {
	FetchOrdersForCustomers(ctx context.Context, customerIDs []int) (map[int][]Order, error)
}

type FakeCustomerRepo struct {
	customers []Customer
}

func (f *FakeCustomerRepo) ListCustomers(ctx context.Context) ([]Customer, error) {
	return f.customers, nil
}

type CountingOrderRepo struct {
	mu sync.Mutex

	orders map[int][]Order

	indvFetchCount int

	batchedSupported bool
}

func NewCountingOrderRepo(orders map[int][]Order, batchedSupported bool) *CountingOrderRepo {
	return &CountingOrderRepo{
		orders:           orders,
		batchedSupported: batchedSupported,
	}
}

func (c *CountingOrderRepo) FetchOrdersForCustomer(ctx context.Context, customerID int) ([]Order, error) {
	c.mu.Lock()
	c.indvFetchCount++
	c.mu.Unlock()

	time.Sleep(1 * time.Millisecond)
	if l, ok := c.orders[customerID]; ok {
		return l, nil
	}
	return nil, nil
}

func (c *CountingOrderRepo) FetchOrdersForCustomers(ctx context.Context, customerIDs []int) (map[int][]Order, error) {
	if !c.batchedSupported {
		out := make(map[int][]Order)
		for _, id := range customerIDs {
			ords, _ := c.FetchOrdersForCustomer(ctx, id)
			out[id] = ords
		}
		return out, nil
	}
	out := make(map[int][]Order)
	for _, id := range customerIDs {
		if l, ok := c.orders[id]; ok {
			out[id] = l
		} else {
			out[id] = []Order{}
		}
	}
	time.Sleep(3 * time.Millisecond)
	return out, nil
}

func (c *CountingOrderRepo) GetIndividualFetchCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.indvFetchCount
}


var ErrNotImplemented = errors.New("not implemented")

func callBuildCustomerSummaries(ctx context.Context, custRepo CustomerRepo, ordRepo OrderRepo) ([]CustomerWithOrders, error) {
	return nil, ErrNotImplemented
}


func Test_HappyPath_BatchedFetchesReduceIndividualCalls(t *testing.T) {
	customers := []Customer{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
		{ID: 3, Name: "Carol"},
		{ID: 4, Name: "Dave"},
		{ID: 5, Name: "Eve"},
	}
	custRepo := &FakeCustomerRepo{customers: customers}
	ordersMap := map[int][]Order{}
	for _, c := range customers {
		ordersMap[c.ID] = []Order{
			{ID: c.ID*10 + 1, CustomerID: c.ID, AmountCents: 100},
			{ID: c.ID*10 + 2, CustomerID: c.ID, AmountCents: 200},
		}
	}
	ordRepo := NewCountingOrderRepo(ordersMap, true) 

	ctx := context.Background()

	out, err := callBuildCustomerSummaries(ctx, custRepo, ordRepo)
	if err != nil {
		t.Fatalf("callBuildCustomerSummaries not implemented or failed: %v", err)
	}
	if len(out) != len(customers) {
		t.Fatalf("expected %d customers in output, got %d", len(customers), len(out))
	}
	for _, item := range out {
		if len(item.Orders) != 2 {
			t.Fatalf("expected 2 orders for customer %d, got %d", item.ID, len(item.Orders))
		}
		expectedTotal := int64(300)
		if item.TotalAmount != expectedTotal {
			t.Fatalf("expected total %d for customer %d, got %d", expectedTotal, item.ID, item.TotalAmount)
		}
	}

	indv := ordRepo.GetIndividualFetchCount()
	if indv > 2 {
		t.Fatalf("expected <= 2 per-customer fetch calls, got %d", indv)
	}
}

func Test_EmptyCustomerList(t *testing.T) {
	custRepo := &FakeCustomerRepo{customers: []Customer{}}
	ordRepo := NewCountingOrderRepo(map[int][]Order{}, true)
	ctx := context.Background()

	out, err := callBuildCustomerSummaries(ctx, custRepo, ordRepo)
	if err != nil {
		t.Fatalf("error calling service: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected empty output for no customers, got %d items", len(out))
	}
}

func Test_MissingOrdersHandledGracefully(t *testing.T) {
	customers := []Customer{{ID: 1, Name: "Solo"}}
	custRepo := &FakeCustomerRepo{customers: customers}
	ordersMap := map[int][]Order{} // no orders for customer
	ordRepo := NewCountingOrderRepo(ordersMap, true)
	ctx := context.Background()

	out, err := callBuildCustomerSummaries(ctx, custRepo, ordRepo)
	if err != nil {
		t.Fatalf("error calling service: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 customer, got %d", len(out))
	}
	if out[0].TotalAmount != 0 {
		t.Fatalf("expected total 0 when no orders exist, got %d", out[0].TotalAmount)
	}
}

func Test_LargeCustomerSet_VerifyCallCountConstraint(t *testing.T) {
	n := 50
	customers := make([]Customer, 0, n)
	ordersMap := map[int][]Order{}
	for i := 1; i <= n; i++ {
		customers = append(customers, Customer{ID: i, Name: "C"})
		ordersMap[i] = []Order{
			{ID: i*10 + 1, CustomerID: i, AmountCents: 10},
		}
	}
	custRepo := &FakeCustomerRepo{customers: customers}
	ordRepo := NewCountingOrderRepo(ordersMap, true)
	ctx := context.Background()

	out, err := callBuildCustomerSummaries(ctx, custRepo, ordRepo)
	if err != nil {
		t.Fatalf("error calling service: %v", err)
	}
	if len(out) != n {
		t.Fatalf("expected %d customers, got %d", n, len(out))
	}
	if ordRepo.GetIndividualFetchCount() > 2 {
		t.Fatalf("expected <=2 individual fetch calls for batched repo, got %d", ordRepo.GetIndividualFetchCount())
	}
}

func Test_Concurrency_Safety(t *testing.T) {
	customers := []Customer{{ID: 1, Name: "A"}, {ID: 2, Name: "B"}, {ID: 3, Name: "C"}}
	custRepo := &FakeCustomerRepo{customers: customers}
	ordersMap := map[int][]Order{
		1: {{ID: 11, CustomerID: 1, AmountCents: 100}},
		2: {{ID: 21, CustomerID: 2, AmountCents: 200}},
		3: {{ID: 31, CustomerID: 3, AmountCents: 300}},
	}
	ordRepo := NewCountingOrderRepo(ordersMap, true)

	ctx := context.Background()
	wg := sync.WaitGroup{}
	errCh := make(chan error, 10)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := callBuildCustomerSummaries(ctx, custRepo, ordRepo)
			if err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for e := range errCh {
		t.Fatalf("concurrent call failed: %v", e)
	}
	if ordRepo.GetIndividualFetchCount() > 2 {
		t.Fatalf("expected <=2 individual fetch calls even under concurrency, got %d", ordRepo.GetIndividualFetchCount())
	}
}

func Test_NegativeAmountsHandled(t *testing.T) {
	customers := []Customer{{ID: 1, Name: "N"}}
	custRepo := &FakeCustomerRepo{customers: customers}
	ordersMap := map[int][]Order{
		1: {
			{ID: 1, CustomerID: 1, AmountCents: -100},
			{ID: 2, CustomerID: 1, AmountCents: 200},
		},
	}
	ordRepo := NewCountingOrderRepo(ordersMap, true)
	ctx := context.Background()

	out, err := callBuildCustomerSummaries(ctx, custRepo, ordRepo)
	if err != nil {
		t.Fatalf("service failed: %v", err)
	}
	if out[0].TotalAmount != 100 {
		t.Fatalf("expected total 100, got %d", out[0].TotalAmount)
	}
}

func Test_BatchedNotSupported_FallsBackAndCounts(t *testing.T) {
	customers := []Customer{{ID: 1, Name: "A"}, {ID: 2, Name: "B"}}
	custRepo := &FakeCustomerRepo{customers: customers}
	ordersMap := map[int][]Order{
		1: {{ID: 11, CustomerID: 1, AmountCents: 100}},
		2: {{ID: 21, CustomerID: 2, AmountCents: 200}},
	}
	ordRepo := NewCountingOrderRepo(ordersMap, false) 
	ctx := context.Background()

	_, err := callBuildCustomerSummaries(ctx, custRepo, ordRepo)
	if err != nil {
		t.Fatalf("service failed: %v", err)
	}
	if ordRepo.GetIndividualFetchCount() < 2 {
		t.Fatalf("expected per-customer fetches when batched not supported; got %d", ordRepo.GetIndividualFetchCount())
	}
}
