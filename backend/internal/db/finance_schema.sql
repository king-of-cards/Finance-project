-- ============================================================
-- King of Cards -- Finance / Procurement Schema
-- All tables prefixed "finance_" to avoid collisions with the
-- existing card-game tables (users, vendors, audit_log, etc.)
-- already present in the "public" schema of this same database.
--
-- Safe to run directly with:
--   psql "$DATABASE_URL" -f db/finance_schema.sql
-- ============================================================

BEGIN;

-- ---------- 1. master / reference tables ----------

CREATE TABLE IF NOT EXISTS finance_users (
    user_id     VARCHAR(20)  PRIMARY KEY,
    name        VARCHAR(120) NOT NULL,
    role        VARCHAR(20)  NOT NULL CHECK (role IN ('finance', 'admin', 'approver')),
    email       VARCHAR(160) UNIQUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS finance_vendors (
    vendor_id           VARCHAR(20)  PRIMARY KEY,          -- e.g. VN-001
    name                VARCHAR(160) NOT NULL,
    city                VARCHAR(80),
    contact_person      VARCHAR(120),
    phone               VARCHAR(30),
    email               VARCHAR(160),
    address             TEXT,
    gst_number          VARCHAR(20),
    pan_number          VARCHAR(15),
    payment_terms       VARCHAR(30) NOT NULL DEFAULT 'Net 30'
        CHECK (payment_terms IN ('Net 15','Net 30','Net 45','Advance 50%','On Delivery')),
    avg_delivery_days   INTEGER     NOT NULL DEFAULT 7,
    rating              SMALLINT    CHECK (rating BETWEEN 1 AND 5),
    status              VARCHAR(20) NOT NULL DEFAULT 'Active'
        CHECK (status IN ('Active','Inactive')),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS finance_charge_types (
    charge_type_id      VARCHAR(30)   PRIMARY KEY,          -- e.g. 'nameplate'
    name                VARCHAR(60)   NOT NULL UNIQUE,       -- 'Nameplate'
    default_rate_min    NUMERIC(10,2) NOT NULL DEFAULT 0,
    default_rate_max    NUMERIC(10,2) NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS finance_app_settings (
    setting_key    VARCHAR(60) PRIMARY KEY,
    setting_value  TEXT        NOT NULL,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---------- 2. transactional core ----------

CREATE TABLE IF NOT EXISTS finance_purchase_orders (
    po_number               VARCHAR(20)  PRIMARY KEY,     -- e.g. PO-2026-1000
    customer_order_no       VARCHAR(20)  NOT NULL,        -- e.g. ORD-5200
    vendor_id                VARCHAR(20)  NOT NULL REFERENCES finance_vendors(vendor_id),
    created_by                VARCHAR(20)  NOT NULL REFERENCES finance_users(user_id),
    approver_id                VARCHAR(20)  REFERENCES finance_users(user_id),

    status                       VARCHAR(30) NOT NULL DEFAULT 'PO Raised'
        CHECK (status IN ('PO Raised','Waiting for Vendor','Received','Verified',
                           'Pending Approval','Approved','Paid','Rejected','Cancelled','Hold')),
    payment_status                 VARCHAR(20) NOT NULL DEFAULT 'Unpaid'
        CHECK (payment_status IN ('Unpaid','Payment Ready','Paid')),

    gst_pct                          NUMERIC(5,2) NOT NULL DEFAULT 18,

    ordered_date                      DATE,
    expected_delivery_date              DATE,
    received_date                        DATE,
    invoice_no                            VARCHAR(40),
    invoice_date                           DATE,
    due_date                                DATE,
    approval_date                            DATE,

    -- financial snapshot from computeEntry() -- frozen once written,
    -- never recomputed live, so approved totals can't drift
    base_total                 NUMERIC(14,2) NOT NULL DEFAULT 0,
    packaging_total            NUMERIC(14,2) NOT NULL DEFAULT 0,
    transport_total            NUMERIC(14,2) NOT NULL DEFAULT 0,
    handling_total             NUMERIC(14,2) NOT NULL DEFAULT 0,
    other_charges_total        NUMERIC(14,2) NOT NULL DEFAULT 0,
    gross_amount               NUMERIC(14,2) NOT NULL DEFAULT 0,
    gst_amount                 NUMERIC(14,2) NOT NULL DEFAULT 0,
    landing_cost               NUMERIC(14,2) NOT NULL DEFAULT 0,  -- final payable
    total_qty                  INTEGER      NOT NULL DEFAULT 0,
    landing_cost_per_unit      NUMERIC(14,2) NOT NULL DEFAULT 0,
    selling_total              NUMERIC(14,2) NOT NULL DEFAULT 0,
    gross_profit               NUMERIC(14,2) NOT NULL DEFAULT 0,
    gross_margin_pct           NUMERIC(6,2)  NOT NULL DEFAULT 0,
    landing_pct                NUMERIC(6,2)  NOT NULL DEFAULT 0,

    remarks         TEXT,
    delivery_note   TEXT,
    locked          BOOLEAN     NOT NULL DEFAULT false,

    created_date    TIMESTAMPTZ NOT NULL DEFAULT now(),
    modified_date   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_po_vendor         ON finance_purchase_orders(vendor_id);
CREATE INDEX IF NOT EXISTS idx_po_status         ON finance_purchase_orders(status);
CREATE INDEX IF NOT EXISTS idx_po_payment_status ON finance_purchase_orders(payment_status);

CREATE TABLE IF NOT EXISTS finance_order_line_items (
    line_item_id             BIGSERIAL     PRIMARY KEY,
    po_number                VARCHAR(20)   NOT NULL REFERENCES finance_purchase_orders(po_number) ON DELETE CASCADE,
    sku_code                 VARCHAR(40),
    product_name             VARCHAR(160)  NOT NULL,
    quantity                 INTEGER       NOT NULL CHECK (quantity > 0),
    rate_per_unit             NUMERIC(10,2) NOT NULL CHECK (rate_per_unit >= 0),
    packaging_flat             NUMERIC(10,2) NOT NULL DEFAULT 0,
    selling_price_per_unit       NUMERIC(10,2) NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_line_items_po ON finance_order_line_items(po_number);

CREATE TABLE IF NOT EXISTS finance_line_item_charges (
    charge_id        BIGSERIAL     PRIMARY KEY,
    line_item_id     BIGINT        NOT NULL REFERENCES finance_order_line_items(line_item_id) ON DELETE CASCADE,
    charge_type_id   VARCHAR(30)   NOT NULL REFERENCES finance_charge_types(charge_type_id),
    rate_per_piece   NUMERIC(10,2) NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_line_charges_item ON finance_line_item_charges(line_item_id);

-- ---------- 3. supporting / audit tables ----------

CREATE TABLE IF NOT EXISTS finance_order_comments (
    comment_id     BIGSERIAL   PRIMARY KEY,
    po_number      VARCHAR(20) NOT NULL REFERENCES finance_purchase_orders(po_number) ON DELETE CASCADE,
    author_id      VARCHAR(20) NOT NULL REFERENCES finance_users(user_id),
    comment_text   TEXT        NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS finance_order_status_history (
    history_id    BIGSERIAL   PRIMARY KEY,
    po_number     VARCHAR(20) NOT NULL REFERENCES finance_purchase_orders(po_number) ON DELETE CASCADE,
    from_status   VARCHAR(30),
    to_status     VARCHAR(30) NOT NULL,
    changed_by    VARCHAR(20) NOT NULL REFERENCES finance_users(user_id),
    note          TEXT,
    changed_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_status_history_po ON finance_order_status_history(po_number);

CREATE TABLE IF NOT EXISTS finance_attachments (
    attachment_id   BIGSERIAL    PRIMARY KEY,
    po_number       VARCHAR(20)  NOT NULL REFERENCES finance_purchase_orders(po_number) ON DELETE CASCADE,
    file_name       VARCHAR(255) NOT NULL,
    file_url        TEXT         NOT NULL,
    uploaded_by     VARCHAR(20)  NOT NULL REFERENCES finance_users(user_id),
    uploaded_at     TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_attachments_po ON finance_attachments(po_number);

COMMIT;

-- ============================================================
-- Reference seed data (safe to re-run — ON CONFLICT DO NOTHING)
-- ============================================================

BEGIN;

INSERT INTO finance_charge_types (charge_type_id, name, default_rate_min, default_rate_max) VALUES
    ('nameplate',      'Nameplate',           1,   6),
    ('ep_sticker',     'EP Sticker',          0.5, 3),
    ('mgi',            'MGI',                 2,   9),
    ('scodix',         'Scodix',              3,   12),
    ('extra_custom',   'Extra Customisation', 2,   15),
    ('transportation', 'Transportation',      1,   8),
    ('handling',       'Handling',            0.5, 4)
ON CONFLICT (charge_type_id) DO NOTHING;

INSERT INTO finance_users (user_id, name, role) VALUES
    ('U-ADMIN', 'Rahul Admin',    'admin'),
    ('U-FIN01', 'Priya Finance',  'finance'),
    ('U-FIN02', 'Amit Finance',   'finance'),
    ('U-FIN03', 'Sneha Finance',  'finance'),
    ('U-FIN04', 'Rohan Finance',  'finance')
ON CONFLICT (user_id) DO NOTHING;

INSERT INTO finance_app_settings (setting_key, setting_value) VALUES
    ('default_gst_pct',     '18'),
    ('currency',             'INR'),
    ('margin_threshold_pct',  '12'),
    ('financial_year',          'FY 2025-26')
ON CONFLICT (setting_key) DO NOTHING;

COMMIT;