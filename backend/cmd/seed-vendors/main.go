package main

import (
"context"
"fmt"
"log"
"time"

"github.com/king-of-cards/finance-project/internal/config"
"github.com/king-of-cards/finance-project/internal/db"
)

type seedVendor struct {
VendorID     string
Name         string
City         string
Status       string
PaymentTerms string
Rating       int
}

type seedPO struct {
PONumber              string
CustomerOrderNo       string
VendorID              string
Status                string
OrderedDaysAgo        int
ReceivedDaysAfterThat int
LandingCost           float64
GrossMarginPct        float64
TotalQty              int
}

func main() {
cfg, err := config.Load()
if err != nil {
log.Fatalf("config error: %v", err)
}

ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

pool, err := db.NewPool(ctx, cfg)
if err != nil {
log.Fatalf("could not connect to database: %v", err)
}
defer pool.Close()

vendors := []seedVendor{
{VendorID: "V001", Name: "Steel Traders Pvt Ltd", City: "Mumbai", Status: "Active", PaymentTerms: "Net 30", Rating: 4},
{VendorID: "V002", Name: "Metal Works India", City: "Pune", Status: "Active", PaymentTerms: "Net 15", Rating: 5},
{VendorID: "V003", Name: "Old Supplier Co", City: "Delhi", Status: "Inactive", PaymentTerms: "Net 45", Rating: 2},
}

for _, v := range vendors {
_, err := pool.Exec(ctx, `
INSERT INTO finance_vendors (vendor_id, name, city, status, payment_terms, rating)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (vendor_id) DO UPDATE
SET name = EXCLUDED.name, city = EXCLUDED.city, status = EXCLUDED.status,
    payment_terms = EXCLUDED.payment_terms, rating = EXCLUDED.rating
`, v.VendorID, v.Name, v.City, v.Status, v.PaymentTerms, v.Rating)
if err != nil {
log.Fatalf("seeding vendor %s: %v", v.VendorID, err)
}
fmt.Printf("seeded vendor: %s (%s)\n", v.VendorID, v.Name)
}

pos := []seedPO{
{PONumber: "PO-1001", CustomerOrderNo: "CO-1001", VendorID: "V001", Status: "Paid", OrderedDaysAgo: 30, ReceivedDaysAfterThat: 5, LandingCost: 50000, GrossMarginPct: 22.5, TotalQty: 100},
{PONumber: "PO-1002", CustomerOrderNo: "CO-1002", VendorID: "V001", Status: "Received", OrderedDaysAgo: 20, ReceivedDaysAfterThat: 4, LandingCost: 32000, GrossMarginPct: 18.0, TotalQty: 60},
{PONumber: "PO-1003", CustomerOrderNo: "CO-1003", VendorID: "V001", Status: "PO Raised", OrderedDaysAgo: 2, ReceivedDaysAfterThat: -1, LandingCost: 15000, GrossMarginPct: 20.0, TotalQty: 30},
{PONumber: "PO-2001", CustomerOrderNo: "CO-2001", VendorID: "V002", Status: "Approved", OrderedDaysAgo: 15, ReceivedDaysAfterThat: 3, LandingCost: 75000, GrossMarginPct: 28.0, TotalQty: 150},
{PONumber: "PO-2002", CustomerOrderNo: "CO-2002", VendorID: "V002", Status: "Verified", OrderedDaysAgo: 10, ReceivedDaysAfterThat: 6, LandingCost: 41000, GrossMarginPct: 25.5, TotalQty: 80},
{PONumber: "PO-2003", CustomerOrderNo: "CO-2003", VendorID: "V002", Status: "Waiting for Vendor", OrderedDaysAgo: 1, ReceivedDaysAfterThat: -1, LandingCost: 20000, GrossMarginPct: 24.0, TotalQty: 40},
{PONumber: "PO-3001", CustomerOrderNo: "CO-3001", VendorID: "V003", Status: "Cancelled", OrderedDaysAgo: 40, ReceivedDaysAfterThat: -1, LandingCost: 10000, GrossMarginPct: 15.0, TotalQty: 20},
{PONumber: "PO-3002", CustomerOrderNo: "CO-3002", VendorID: "V003", Status: "Rejected", OrderedDaysAgo: 35, ReceivedDaysAfterThat: -1, LandingCost: 12000, GrossMarginPct: 10.0, TotalQty: 25},
}

for _, p := range pos {
orderedDate := time.Now().AddDate(0, 0, -p.OrderedDaysAgo)

var receivedDate *time.Time
if p.ReceivedDaysAfterThat >= 0 {
d := orderedDate.AddDate(0, 0, p.ReceivedDaysAfterThat)
receivedDate = &d
}

_, err := pool.Exec(ctx, `
INSERT INTO finance_purchase_orders
(po_number, customer_order_no, vendor_id, created_by, status,
 ordered_date, received_date, landing_cost, gross_margin_pct, total_qty)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (po_number) DO UPDATE
SET status = EXCLUDED.status,
    ordered_date = EXCLUDED.ordered_date,
    received_date = EXCLUDED.received_date,
    landing_cost = EXCLUDED.landing_cost,
    gross_margin_pct = EXCLUDED.gross_margin_pct,
    total_qty = EXCLUDED.total_qty
`, p.PONumber, p.CustomerOrderNo, p.VendorID, "admin", p.Status,
orderedDate, receivedDate, p.LandingCost, p.GrossMarginPct, p.TotalQty)
if err != nil {
log.Fatalf("seeding PO %s: %v", p.PONumber, err)
}
fmt.Printf("seeded PO: %s (%s, %s)\n", p.PONumber, p.VendorID, p.Status)
}

fmt.Println("done.")
}
