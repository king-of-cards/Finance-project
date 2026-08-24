use strict;
use warnings;
use HTTP::Tiny;
use JSON::PP;

my $token = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiYWRtaW4iLCJyb2xlIjoiYXBwcm92ZXIiLCJleHAiOjE3ODc2NDgxOTgsImlhdCI6MTc4NzU2MTc5OCwianRpIjoiZDE4OWU0OTctZGJmNi00ODY1LWJkNTktYzk0MThkNjFhM2ViIn0.KOX_zAWpMReLz_G6g-HWyv4pZjlEHeGIg1oPPXS55NM';

my $body = {
    vendor_id          => 'V001',
    customer_order_no  => 'CO-TEST-1',
    gst_pct            => 18,
    skus => [
        {
            product_name           => 'Test Widget',
            quantity                => 10,
            rate_per_unit           => 100,
            packaging_flat          => 5,
            selling_price_per_unit  => 150,
        }
    ]
};

my $json_body = encode_json($body);

my $response = HTTP::Tiny->new->post(
    'http://localhost:8080/api/purchase-orders',
    {
        headers => {
            'Content-Type'  => 'application/json',
            'Authorization' => "Bearer $token",
        },
        content => $json_body,
    }
);

print "HTTP STATUS: $response->{status}\n";
print "RESPONSE BODY:\n";
print $response->{content} . "\n";

if ($response->{success}) {
    my $decoded = decode_json($response->{content});
    print "PO Number created: $decoded->{po_number}\n" if $decoded->{po_number};
} else {
    print "Request failed.\n";
}