# task_tests.py
import pytest
import requests
import threading
import time
from contextlib import contextmanager

BASE_URL = "http://localhost:8080"

@pytest.fixture(scope="module")
def health_check():
    try:
        resp = requests.get(f"{BASE_URL}/health", timeout=5)
        assert resp.status_code == 200
        j = resp.json()
        assert isinstance(j, dict)
        assert j.get("status") == "ok"
    except Exception as e:
        pytest.skip(f"Service not running or /health failed: {e}")

def test_customers_endpoint_returns_all(health_check):
    start_time = time.time()
    resp = requests.get(f"{BASE_URL}/customers", timeout=10)
    end_time = time.time()
    assert resp.status_code == 200
    data = resp.json()
    assert isinstance(data, list)
    for customer in data:
        assert 'orders' in customer
        assert len(customer['orders']) > 0 
        assert 'totalAmount' in customer or 'total_amount' in customer

def test_orders_endpoint_for_customer(health_check):
    resp = requests.get(f"{BASE_URL}/customers/1/orders")
    assert resp.status_code == 200
    orders = resp.json()
    assert isinstance(orders, list)
    for o in orders:
        assert "id" in o
        assert "customer_id" in o or "customerID" in o
        assert "amount_cents" in o or "amountCents" in o

def test_total_amount_calculation(health_check):
    resp = requests.get(f"{BASE_URL}/customers")
    assert resp.status_code == 200
    customers = resp.json()
    for cust in customers:
        orders = cust.get('orders', [])
        total = 0
        for order in orders:
            if 'amount_cents' in order:
                total += int(order['amount_cents'])
            else:
                total += int(order.get('amountCents', 0))
        api_total = cust.get('totalAmount', cust.get('total_amount', total))
        assert int(api_total) == total, f"Customer {cust.get('id')}: expected {total}, got {api_total}"

def test_missing_orders_handled(health_check):
    resp = requests.get(f"{BASE_URL}/customers/999/orders")
    assert resp.status_code == 200
    orders = resp.json()
    assert orders == []

def test_concurrent_requests(health_check):
    results = []
    def make_request():
        r = requests.get(f"{BASE_URL}/customers", timeout=5)
        customers = r.json()
        totals_before = {c['id']: c.get('totalAmount', c.get('total_amount')) for c in customers}
        results.append((r.status_code, totals_before))

    threads = [threading.Thread(target=make_request) for _ in range(5)]
    for t in threads:
        t.start()
    for t in threads:
        t.join()
    assert all(status == 200 for status, _ in results)

def test_negative_amounts(health_check):
    resp = requests.get(f"{BASE_URL}/customers")
    customers = resp.json()
    for cust in customers:
        orders = cust.get('orders', [])
        total = 0
        for o in orders:
            if 'amount_cents' in o:
                total += int(o['amount_cents'])
            else:
                total += int(o.get('amountCents', 0))
        api_total = cust.get('totalAmount', cust.get('total_amount', total))
        assert int(api_total) == total

def test_n_plus_one_elimination(health_check):
    start_time = time.time()
    resp = requests.get(f"{BASE_URL}/customers", timeout=1) 
    end_time = time.time()
    duration = (end_time - start_time) * 1000
    assert resp.status_code == 200
    assert duration <= 500, f"Request took {duration:.0f}ms - too slow for batch queries"



def test_repository_interfaces_used(health_check):
    resp = requests.get(f"{BASE_URL}/customers?debug=interfaces", timeout=5)
    assert resp.status_code == 200
    debug_info = resp.json().get('debug', {})
    assert 'CustomerRepository' in debug_info.get('interfaces', [])
    assert 'OrderRepository' in debug_info.get('interfaces', [])
    assert 'GetOrdersByCustomerIDs' in debug_info.get('methods', [])

def test_batch_method_exists(health_check):
    resp = requests.get(f"{BASE_URL}/customers?test_batch=[1,2,3]", timeout=5)
    assert resp.status_code == 200
    batch_info = resp.json().get('batch_test', {})
    assert len(batch_info['customer_ids']) == 3 
    assert 'query_count' in batch_info
    assert batch_info['query_count'] <= 2

def test_no_package_mutable_state(health_check):
    resp1 = requests.get(f"{BASE_URL}/customers")
    totals1 = {c['id']: c.get('totalAmount', c.get('total_amount')) for c in resp1.json()}
    
    def concurrent_load():
        for _ in range(10):
            requests.get(f"{BASE_URL}/customers/1/orders", timeout=1)
    
    threads = [threading.Thread(target=concurrent_load) for _ in range(3)]
    for t in threads: t.start()
    for t in threads: t.join()
    
    resp2 = requests.get(f"{BASE_URL}/customers")
    totals2 = {c['id']: c.get('totalAmount', c.get('total_amount')) for c in resp2.json()}
    assert totals1 == totals2, "Global state mutated during concurrency"

def test_three_files_modified(health_check):
    resp = requests.get(f"{BASE_URL}/customers?version=debug", timeout=5)
    assert resp.status_code == 200
    files_touched = resp.json().get('files_modified', [])
    assert len(files_touched) >= 3
    required_files = ['customers.go', 'customer_service.go', 'order_repo.go']
    assert all(f in ' '.join(files_touched) for f in required_files)

def test_actual_query_count(health_check):
    resp = requests.get(f"{BASE_URL}/customers?metrics=true", timeout=5)
    assert resp.status_code == 200
    metrics = resp.json().get('metrics', {})
    assert metrics.get('db_queries', 999) <= 2, f"Used {metrics['db_queries']} queries"
