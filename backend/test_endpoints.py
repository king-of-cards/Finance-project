"""
Quick smoke test for the purchase-order API.

Requires: Python 3.8+, and the `requests` package:
    pip install requests

Usage:
    python test_endpoints.py
    python test_endpoints.py --base-url http://localhost:8080 --user-id admin --password secret
    python test_endpoints.py --vendor-id V-001
"""

import argparse
import sys
import time

import requests

PASS = 0
FAIL = 0


def check(description, expected_status, method="GET", url="", headers=None, json_body=None):
    """Make a request, print PASS/FAIL, and return the parsed JSON (or raw text)."""
    global PASS, FAIL
    headers = headers or {}

    try:
        resp = requests.request(method, url, headers=headers, json=json_body, timeout=15)
        status = resp.status_code
    except requests.exceptions.RequestException as e:
        print(f"FAIL  [no response] {description}")
        print(f"      {e}")
        FAIL += 1
        return None

    if status == expected_status:
        print(f"PASS  [{status}] {description}")
        PASS += 1
    else:
        print(f"FAIL  [{status}, expected {expected_status}] {description}")
        print(f"      {resp.text[:500]}")
        FAIL += 1

    try:
        return resp.json()
    except ValueError:
        return resp.text


def main():
    parser = argparse.ArgumentParser(description="Smoke test the purchase-order API")
    parser.add_argument("--base-url", default="http://localhost:8080")
    parser.add_argument("--user-id", default="admin", help="login identifier (the API calls this user_id)")
    parser.add_argument("--password", default="changeme")
    parser.add_argument("--vendor-id", default="")
    args = parser.parse_args()

    base_url = args.base_url.rstrip("/")

    print("== 1. Login ==")
    login_body = check(
        "POST /api/login",
        200,
        method="POST",
        url=f"{base_url}/api/login",
        json_body={"user_id": args.user_id, "password": args.password},
    )

    token = None
    if isinstance(login_body, dict):
        token = login_body.get("token") or login_body.get("access_token")

    if not token:
        print("!! Could not find a token field in the login response — check the body below")
        print("!! and adjust the .get('token') / .get('access_token') lookup in this script.")
        print(login_body)
        sys.exit(1)

    auth_headers = {"Authorization": f"Bearer {token}"}

    print("\n== 2. Auth sanity checks ==")
    check("GET /api/auth/me", 200, url=f"{base_url}/api/auth/me", headers=auth_headers)
    check(
        "GET /api/purchase-orders without token -> should be 401",
        401,
        url=f"{base_url}/api/purchase-orders",
    )

    print("\n== 3. List purchase orders ==")
    check(
        "GET /api/purchase-orders",
        200,
        url=f"{base_url}/api/purchase-orders?page=1&limit=10",
        headers=auth_headers,
    )
    check(
        "GET /api/purchase-orders with filters",
        200,
        url=f"{base_url}/api/purchase-orders?status=PO+Raised&payment_status=Unpaid",
        headers=auth_headers,
    )

    print("\n== 4. Vendors (needed to create a PO) ==")
    vendors_body = check(
        "GET /api/vendors",
        200,
        url=f"{base_url}/api/vendors?limit=1",
        headers=auth_headers,
    )

    vendor_id = args.vendor_id
    if not vendor_id and isinstance(vendors_body, dict):
        vendors = vendors_body.get("vendors") or []
        if vendors:
            vendor_id = vendors[0].get("vendor_id")

    print("\n== 5. Create a purchase order ==")
    po_number = None
    if vendor_id:
        create_payload = {
            "vendor_id": vendor_id,
            "customer_order_no": f"TEST-ORDER-{int(time.time())}",
            "gst_pct": 18,
            "skus": [
                {
                    "product_name": "Test Widget",
                    "quantity": 10,
                    "rate_per_unit": 100,
                    "packaging_flat": 5,
                    "selling_price_per_unit": 150,
                    "charges": [{"charge_type_id": "FREIGHT", "rate_per_piece": 2}],
                }
            ],
        }
        create_body = check(
            "POST /api/purchase-orders",
            201,
            method="POST",
            url=f"{base_url}/api/purchase-orders",
            headers=auth_headers,
            json_body=create_payload,
        )
        if isinstance(create_body, dict):
            po_number = create_body.get("po_number")
    else:
        print("SKIP  no vendor_id available — pass --vendor-id V-001 to test creation")

    print("\n== 6. Get purchase order detail ==")
    if po_number:
        check(
            f"GET /api/purchase-orders/{po_number}",
            200,
            url=f"{base_url}/api/purchase-orders/{po_number}",
            headers=auth_headers,
        )
    else:
        print("SKIP  no po_number from creation step")

    check(
        "GET /api/purchase-orders/DOES-NOT-EXIST -> should be 404",
        404,
        url=f"{base_url}/api/purchase-orders/DOES-NOT-EXIST",
        headers=auth_headers,
    )

    print("\n== 7. Export CSV ==")
    global PASS, FAIL
    try:
        resp = requests.get(f"{base_url}/api/purchase-orders/export", headers=auth_headers, timeout=15)
        if resp.status_code == 200:
            with open("purchase_orders_export.csv", "wb") as f:
                f.write(resp.content)
            print("PASS  [200] export saved to purchase_orders_export.csv")
            PASS += 1
            preview = resp.content.decode(errors="replace").splitlines()[:3]
            print("\n".join(preview))
        else:
            print(f"FAIL  [{resp.status_code}] export request failed")
            print(resp.text[:500])
            FAIL += 1
    except requests.exceptions.RequestException as e:
        print(f"FAIL  export request errored: {e}")
        FAIL += 1

    print("\n== 8. Logout ==")
    check("POST /api/logout", 200, method="POST", url=f"{base_url}/api/logout", headers=auth_headers)

    print("\n==================================")
    print(f"Passed: {PASS}   Failed: {FAIL}")
    print("==================================")

    sys.exit(1 if FAIL else 0)


if __name__ == "__main__":
    main()