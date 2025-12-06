package perf001_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// --- Types from the base repo (shims / expected shapes) ---
// Your base repo probably already defines similar types. Tests assume the following.
// If your base repo uses different names or packages, adapt the imports in your repo's test runner
// or provide small adapter functions in your code so the handler/service under test can be invoked.

type Customer struct {
	ID   int
	Name string
}

type Order struct {
	ID         int
	CustomerID int
	AmountCents int64
}

// Expected API response item shape for assertion (can be adapted to actual project JSON)
type CustomerWithOrders struct {
	ID          int
	Name        string
	Orders      []Order
	TotalAmount int64
}

// --- Repository interfaces that the base repo should implement/adhere to ---
type CustomerRepo interface {
	ListCustomers(ctx context.Context) ([]Customer, error)
}

type OrderRepo interface {
	// Existing naive API in base repo may have:
	// FetchOrdersForCustomer(ctx context.Context, customerID int) ([]Order, error)
	// Task requires adding a batched API:
	FetchOrdersForCustomers(ctx context.Context, customerIDs []int) (map[int][]Order, error)
}

// --- Fake repo implementations for tests ---
// FakeCustomerRepo provides a deterministic list of customers.
type FakeCustomerRepo struct {
	customers []Customer
}

func (f *FakeCustomerRepo) ListCustomers(ctx context.Context) ([]Customer, error) {
	return f.customers, nil
}

// CountingOrderRepo implements OrderRepo and counts how many times the per-customer fetch function is invoked.
// It supports both batched and per-customer paths for compatibility.
type CountingOrderRepo struct {
	mu sync.Mutex

	// dataset: map customerID -> orders
	orders map[int][]Order

	// count of individual fetch calls
	indvFetchCount int

	// if set, this repo reports that batched API is NOT implemented (simulate old base)
	batchedSupported bool
}

func NewCountingOrderRepo(orders map[int][]Order, batchedSupported bool) *CountingOrderRepo {
	return &CountingOrderRepo{
		orders:           orders,
		batchedSupported: batchedSupported,
	}
}

// Simulates legacy per-customer fetch (should be avoided by optimized solution)
func (c *CountingOrderRepo) FetchOrdersForCustomer(ctx context.Context, customerID int) ([]Order, error) {
	c.mu.Lock()
	c.indvFetchCount++
	c.mu.Unlock()

	// simulate small DB latency deterministically
	time.Sleep(1 * time.Millisecond)
	if l, ok := c.orders[customerID]; ok {
		return l, nil
	}
	return nil, nil
}

// New method required for optimization
func (c *CountingOrderRepo) FetchOrdersForCustomers(ctx context.Context, customerIDs []int) (map[int][]Order, error) {
	if !c.batchedSupported {
		// If batched not supported, fall back to per-customer calls to mimic legacy behavior.
		out := make(map[int][]Order)
		for _, id := range customerIDs {
			ords, _ := c.FetchOrdersForCustomer(ctx, id)
			out[id] = ords
		}
		return out, nil
	}
	// When batchedSupported is true, return the dataset in a single call (no per-customer increments)
	out := make(map[int][]Order)
	for _, id := range customerIDs {
		if l, ok := c.orders[id]; ok {
			out[id] = l
		} else {
			out[id] = []Order{}
		}
	}
	// Simulate a slightly larger single batched DB latency but far smaller than N per-customer calls
	time.Sleep(3 * time.Millisecond)
	return out, nil
}

func (c *CountingOrderRepo) GetIndividualFetchCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.indvFetchCount
}

// --- Service under test interface ---
// The base repo should expose a function we can invoke in tests. We'll assume a function with signature:
// func BuildCustomerSummaries(ctx context.Context, custRepo CustomerRepo, ordRepo OrderRepo) ([]CustomerWithOrders, error)
// The test harness will call that function. The project implementer should ensure their production code exposes
// a testable function with that signature or adapt their code to provide an adapter.

var ErrNotImplemented = errors.New("not implemented")

// Helper to call the service function under test. Adjust the import path if your repo exposes it elsewhere.
func callBuildCustomerSummaries(ctx context.Context, custRepo CustomerRepo, ordRepo OrderRepo) ([]CustomerWithOrders, error) {
	// Adapter: attempt to call package function if available.
	// To keep tests independent of repo package name, attempt to resolve dynamically is not possible in go tests.
	// Instead: implementers should provide a function at package "perfservice" or update this test import.
	return nil, ErrNotImplemented
}

// --- Tests start here ---

// NOTE: The developer implementing the task must either:
// 1) ensure the repo exposes a function with the signature used in callBuildCustomerSummaries,
// OR
// 2) replace callBuildCustomerSummaries with a direct call to their package/function (preferred).
//
// The tests below are written to be generic; adapt the import to point at your service implementation.

func Test_HappyPath_BatchedFetchesReduceIndividualCalls(t *testing.T) {
	// Setup dataset: 5 customers, each with 2 orders
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
	ordRepo := NewCountingOrderRepo(ordersMap, true) // batched supported

	ctx := context.Background()

	out, err := callBuildCustomerSummaries(ctx, custRepo, ordRepo)
	if err != nil {
		// If the test harness isn't wired to actual implementation, fail with explanatory message.
		t.Fatalf("callBuildCustomerSummaries not implemented or failed: %v", err)
	}

	// correctness checks
	if len(out) != len(customers) {
		t.Fatalf("expected %d customers in output, got %d", len(customers), len(out))
	}
	// Verify totals and orders presence
	for _, item := range out {
		if len(item.Orders) != 2 {
			t.Fatalf("expected 2 orders for customer %d, got %d", item.ID, len(item.Orders))
		}
		expectedTotal := int64(300)
		if item.TotalAmount != expectedTotal {
			t.Fatalf("expected total %d for customer %d, got %d", expectedTotal, item.ID, item.TotalAmount)
		}
	}

	// performance correctness: because batchedSupported=true, the implementation should use the batched method
	// and thus not call the per-customer fetch more than 2 times overall.
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
	// 50 customers - tests batched behavior; per-customer fetch count must remain <= 2
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
	// Ensure multiple concurrent calls do not cause race conditions and that performance constraints hold.
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
	// ensure per-customer fetch count is still constrained
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
	// When batchedSupported=false, the repo falls back to per-customer calls and individual fetch count will increase.
	customers := []Customer{{ID: 1, Name: "A"}, {ID: 2, Name: "B"}}
	custRepo := &FakeCustomerRepo{customers: customers}
	ordersMap := map[int][]Order{
		1: {{ID: 11, CustomerID: 1, AmountCents: 100}},
		2: {{ID: 21, CustomerID: 2, AmountCents: 200}},
	}
	ordRepo := NewCountingOrderRepo(ordersMap, false) // batched NOT supported
	ctx := context.Background()

	_, err := callBuildCustomerSummaries(ctx, custRepo, ordRepo)
	if err != nil {
		t.Fatalf("service failed: %v", err)
	}
	if ordRepo.GetIndividualFetchCount() < 2 {
		t.Fatalf("expected per-customer fetches when batched not supported; got %d", ordRepo.GetIndividualFetchCount())
	}
}
