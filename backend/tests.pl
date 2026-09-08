#!/usr/bin/perl
use strict;
use warnings;
use HTTP::Tiny;
use JSON::PP;

my $BASE    = "http://localhost:8080/api";
my $http    = HTTP::Tiny->new;
my $json    = JSON::PP->new->utf8->canonical;
my $pass_ct = 0;
my $fail_ct = 0;

sub log_result {
    my ($name, $ok, $detail) = @_;
    if ($ok) {
        print "[PASS] $name\n";
        $pass_ct++;
    } else {
        print "[FAIL] $name -- $detail\n";
        $fail_ct++;
    }
}

sub req {
    my (%args) = @_;
    my $method  = $args{method}  // 'GET';
    my $path    = $args{path};
    my $token   = $args{token};
    my $body    = $args{body};

    my %headers = ();
    $headers{Authorization} = "Bearer $token" if $token;

    my %opts = (headers => \%headers);
    if (defined $body) {
        $headers{'Content-Type'} = 'application/json';
        $opts{content} = $json->encode($body);
    }

    my $res = $http->request($method, "$BASE$path", \%opts);
    my $data;
    eval { $data = $json->decode($res->{content}) } if $res->{content};
    return ($res, $data);
}

# ---------------------------------------------------------
# 1. Login
# ---------------------------------------------------------
my ($res, $data) = req(
    method => 'POST',
    path   => '/login',
    body   => { user_id => 'admin', password => 'King@123' },
);
log_result("Login (valid)", $res->{status} == 200 && $data->{token}, "status=$res->{status}");
my $token = $data->{token} // '';

my ($res2, $data2) = req(
    method => 'POST',
    path   => '/login',
    body   => { user_id => 'admin', password => 'WrongPassword' },
);
log_result("Login (invalid password)", $res2->{status} == 401, "status=$res2->{status}");

# ---------------------------------------------------------
# 2. Auth / Me
# ---------------------------------------------------------
($res, $data) = req(path => '/auth/me', token => $token);
log_result("GET /auth/me", $res->{status} == 200 && $data->{user_id} eq 'admin', "status=$res->{status}");

# ---------------------------------------------------------
# 3. Users
# ---------------------------------------------------------
($res, $data) = req(path => '/users', token => $token);
log_result("GET /users", $res->{status} == 200, "status=$res->{status}");

($res, $data) = req(path => '/finance-users?role=finance', token => $token);
log_result("GET /finance-users?role=finance", $res->{status} == 200, "status=$res->{status}");

($res, $data) = req(path => '/finance-users?role=bogus', token => $token);
log_result("GET /finance-users?role=bogus (should 400)", $res->{status} == 400, "status=$res->{status}");

# ---------------------------------------------------------
# 4. Vendors
# ---------------------------------------------------------
($res, $data) = req(path => '/vendors', token => $token);
log_result("GET /vendors", $res->{status} == 200, "status=$res->{status}");

($res, $data) = req(path => '/vendors?status=Active', token => $token);
log_result("GET /vendors?status=Active", $res->{status} == 200, "status=$res->{status}");

($res, $data) = req(path => '/vendors/V001', token => $token);
log_result("GET /vendors/V001", $res->{status} == 200 || $res->{status} == 404, "status=$res->{status}");

($res, $data) = req(path => '/vendors/DOES-NOT-EXIST', token => $token);
log_result("GET /vendors/DOES-NOT-EXIST (should 404)", $res->{status} == 404, "status=$res->{status}");

($res, $data) = req(path => '/vendors/V001/orders', token => $token);
log_result("GET /vendors/V001/orders", $res->{status} == 200, "status=$res->{status}");

($res, $data) = req(
    method => 'POST',
    path   => '/vendors',
    token  => $token,
    body   => { name => "Perl Test Vendor", payment_terms => "Net 30", city => "Testville" },
);
log_result("POST /vendors (create)", $res->{status} == 201, "status=$res->{status}");
my $new_vendor_id = $data->{vendor_id};

if ($new_vendor_id) {
    ($res, $data) = req(
        method => 'PATCH',
        path   => "/vendors/$new_vendor_id",
        token  => $token,
        body   => { phone => "9999999999" },
    );
    log_result("PATCH /vendors/$new_vendor_id", $res->{status} == 200, "status=$res->{status}");

    ($res, $data) = req(method => 'DELETE', path => "/vendors/$new_vendor_id", token => $token);
    log_result("DELETE /vendors/$new_vendor_id (deactivate)", $res->{status} == 200, "status=$res->{status}");
}

# ---------------------------------------------------------
# 5. Purchase Orders
# ---------------------------------------------------------
($res, $data) = req(path => '/purchase-orders', token => $token);
log_result("GET /purchase-orders", $res->{status} == 200, "status=$res->{status}");

($res, $data) = req(path => '/purchase-orders?vendorId=V001', token => $token);
log_result("GET /purchase-orders?vendorId=V001", $res->{status} == 200, "status=$res->{status}");

($res, $data) = req(path => '/purchase-orders?status=bogus', token => $token);
log_result("GET /purchase-orders?status=bogus (should 400)", $res->{status} == 400, "status=$res->{status}");

($res, $data) = req(path => '/purchase-orders/PO-1001', token => $token);
log_result("GET /purchase-orders/PO-1001", $res->{status} == 200, "status=$res->{status}");

($res, $data) = req(path => '/purchase-orders/PO-9999', token => $token);
log_result("GET /purchase-orders/PO-9999 (should 404)", $res->{status} == 404, "status=$res->{status}");

($res, $data) = req(path => '/purchase-orders/PO-1001/status-history', token => $token);
log_result("GET /purchase-orders/PO-1001/status-history", $res->{status} == 200, "status=$res->{status}");

($res, $data) = req(path => '/purchase-orders/export', token => $token);
log_result("GET /purchase-orders/export", $res->{status} == 200, "status=$res->{status}");

# Create a fresh PO to safely test approve/reject/hold on
($res, $data) = req(
    method => 'POST',
    path   => '/purchase-orders',
    token  => $token,
    body   => {
        customer_order_no => "CO-PERL-TEST",
        vendor_id         => "V001",
        gst_pct           => 18,
        skus              => [
            {
                product_name   => "Perl Test SKU",
                quantity       => 10,
                rate_per_unit  => 100,
                packaging_flat => 5,
                charges        => [ { charge_type_id => "handling", rate_per_piece => 2 } ],
            }
        ],
    },
);
log_result("POST /purchase-orders (create)", $res->{status} == 201, "status=$res->{status}");
my $new_po = $data->{po_number};

if ($new_po) {
    my $li_id = $data->{line_items}[0]{line_item_id};

    ($res, $data) = req(
        method => 'PATCH',
        path   => "/purchase-orders/$new_po",
        token  => $token,
        body   => { remarks => "updated via perl test" },
    );
    log_result("PATCH /purchase-orders/$new_po", $res->{status} == 200, "status=$res->{status}");

    ($res, $data) = req(
        method => 'POST',
        path   => "/purchase-orders/$new_po/hold",
        token  => $token,
        body   => { comment => "holding for test" },
    );
    log_result("POST /purchase-orders/$new_po/hold", $res->{status} == 200, "status=$res->{status}");

    ($res, $data) = req(
        method => 'POST',
        path   => "/purchase-orders/$new_po/request-changes",
        token  => $token,
        body   => { comment => "please fix" },
    );
    log_result("POST .../request-changes", $res->{status} == 200, "status=$res->{status}");

    ($res, $data) = req(
        method => 'POST',
        path   => "/purchase-orders/$new_po/approve",
        token  => $token,
        body   => { verifiedLineItemIds => [], comment => "should fail - not all verified" },
    );
    log_result("POST .../approve (should 400, missing line items)", $res->{status} == 400, "status=$res->{status}");

    ($res, $data) = req(
        method => 'POST',
        path   => "/purchase-orders/$new_po/approve",
        token  => $token,
        body   => { verifiedLineItemIds => [$li_id], comment => "approved via perl test" },
    );
    log_result("POST .../approve (valid)", $res->{status} == 200, "status=$res->{status}");

    ($res, $data) = req(
        method => 'POST',
        path   => "/purchase-orders/$new_po/mark-paid",
        token  => $token,
        body   => { comment => "paid via perl test" },
    );
    log_result("POST .../mark-paid", $res->{status} == 200, "status=$res->{status}");

    ($res, $data) = req(method => 'DELETE', path => "/purchase-orders/$new_po", token => $token);
    log_result("DELETE /purchase-orders/$new_po (should 409, locked)", $res->{status} == 409, "status=$res->{status}");
}



# ---------------------------------------------------------
# 5b. Line Items + Charges (standalone access)
# ---------------------------------------------------------

# Create a dedicated PO for line-item/charge testing (needs 2 SKUs so we can
# safely test deleting one without hitting the "last line item" guard)
($res, $data) = req(
    method => 'POST',
    path   => '/purchase-orders',
    token  => $token,
    body   => {
        customer_order_no => "CO-LINEITEM-TEST",
        vendor_id         => "V001",
        gst_pct           => 18,
        skus              => [
            {
                product_name   => "Line Item Test SKU A",
                quantity       => 5,
                rate_per_unit  => 50,
                packaging_flat => 2,
                charges        => [ { charge_type_id => "handling", rate_per_piece => 1 } ],
            },
            {
                product_name   => "Line Item Test SKU B",
                quantity       => 5,
                rate_per_unit  => 60,
                packaging_flat => 2,
            },
        ],
    },
);
log_result("POST /purchase-orders (for line-item tests)", $res->{status} == 201, "status=$res->{status}");
my $li_po = $data->{po_number};
my $li_a  = $data->{line_items}[0]{line_item_id};
my $li_b  = $data->{line_items}[1]{line_item_id};

if ($li_po) {
    ($res, $data) = req(path => "/purchase-orders/$li_po/line-items", token => $token);
    log_result("GET /purchase-orders/$li_po/line-items", $res->{status} == 200, "status=$res->{status}");

    ($res, $data) = req(
        method => 'POST',
        path   => "/purchase-orders/$li_po/line-items",
        token  => $token,
        body   => { product_name => "Added SKU C", quantity => 3, rate_per_unit => 40, packaging_flat => 1 },
    );
    log_result("POST /purchase-orders/$li_po/line-items (add)", $res->{status} == 201, "status=$res->{status}");
    my $li_c = $data->{line_item_id};

    ($res, $data) = req(
        method => 'PATCH',
        path   => "/line-items/$li_a",
        token  => $token,
        body   => { quantity => 10 },
    );
    log_result("PATCH /line-items/$li_a", $res->{status} == 200, "status=$res->{status}");

    ($res, $data) = req(path => "/line-items/$li_a/charges", token => $token);
    log_result("GET /line-items/$li_a/charges", $res->{status} == 200, "status=$res->{status}");

    ($res, $data) = req(
        method => 'PUT',
        path   => "/line-items/$li_a/charges",
        token  => $token,
        body   => { charges => [ { charge_type_id => "transportation", rate_per_piece => 3 } ] },
    );
    log_result("PUT /line-items/$li_a/charges (replace)", $res->{status} == 200, "status=$res->{status}");
    my $new_charge_id = $data->{charges}[0]{charge_id};

    if ($new_charge_id) {
        ($res, $data) = req(method => 'DELETE', path => "/line-items/$li_a/charges/$new_charge_id", token => $token);
        log_result("DELETE /line-items/$li_a/charges/$new_charge_id", $res->{status} == 200, "status=$res->{status}");
    }

    ($res, $data) = req(method => 'DELETE', path => "/line-items/$li_c", token => $token);
    log_result("DELETE /line-items/$li_c (one of three, should succeed)", $res->{status} == 200, "status=$res->{status}");

    ($res, $data) = req(method => 'DELETE', path => "/line-items/$li_b", token => $token);
    log_result("DELETE /line-items/$li_b (down to one left, should succeed)", $res->{status} == 200, "status=$res->{status}");

    ($res, $data) = req(method => 'DELETE', path => "/line-items/$li_a", token => $token);
    log_result("DELETE /line-items/$li_a (last remaining, should 400)", $res->{status} == 400, "status=$res->{status}");
}

# ---------------------------------------------------------
# 5c. Charge Types
# ---------------------------------------------------------
($res, $data) = req(path => '/charge-types', token => $token);
log_result("GET /charge-types", $res->{status} == 200, "status=$res->{status}");

($res, $data) = req(
    method => 'POST',
    path   => '/charge-types',
    token  => $token,
    body   => { charge_type_id => "perl_test_ct", name => "Perl Test Charge", default_rate_min => 1, default_rate_max => 5 },
);
log_result("POST /charge-types (approver, should succeed)", $res->{status} == 201, "status=$res->{status}");

if ($res->{status} == 201) {
    ($res, $data) = req(
        method => 'PATCH',
        path   => "/charge-types/perl_test_ct",
        token  => $token,
        body   => { name => "Perl Test Charge Updated" },
    );
    log_result("PATCH /charge-types/perl_test_ct", $res->{status} == 200, "status=$res->{status}");

    ($res, $data) = req(method => 'DELETE', path => "/charge-types/perl_test_ct", token => $token);
    log_result("DELETE /charge-types/perl_test_ct (unused, should succeed)", $res->{status} == 200, "status=$res->{status}");
}

($res, $data) = req(method => 'DELETE', path => "/charge-types/handling", token => $token);
log_result("DELETE /charge-types/handling (in use, should 409)", $res->{status} == 409, "status=$res->{status}");




# ---------------------------------------------------------
# 6. Logout + revoked token check
# ---------------------------------------------------------
($res, $data) = req(method => 'POST', path => '/logout', token => $token);
log_result("POST /logout", $res->{status} == 200, "status=$res->{status}");

($res, $data) = req(path => '/auth/me', token => $token);
log_result("GET /auth/me after logout (should 401)", $res->{status} == 401, "status=$res->{status}");

# ---------------------------------------------------------
print "\n===================================\n";
print "PASSED: $pass_ct   FAILED: $fail_ct\n";
print "===================================\n";
exit($fail_ct > 0 ? 1 : 0);