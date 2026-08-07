package db

import (
	"strings"
	"testing"
)

func TestBuildVendorQuery_NoFilters(t *testing.T) {
	query, args := buildVendorQuery(VendorFilters{})
	if strings.Contains(query, "WHERE") {
		t.Errorf("expected no WHERE clause when no filters given, got: %s", query)
	}
	if len(args) != 0 {
		t.Errorf("expected 0 args when no filters given, got %d", len(args))
	}

}

func TestBuildVendorQuery_StatusOnly(t *testing.T) {
	query, args := buildVendorQuery(VendorFilters{Status: "Active"})
	if !strings.Contains(query, "status = $1") {
		t.Errorf("expected 'status = $1' in query, got: %s", query)
	}
	if len(args) != 1 || args[0] != "Active" {
		t.Errorf("expected args = [\"Active\"], got %v", args)

	}

}

func TestBuildVendorQuery_SearchOnly(t *testing.T) {
	query, args := buildVendorQuery(VendorFilters{Search: "steel"})

	if !strings.Contains(query, "name ILIKE $1") {
		t.Errorf("expected 'name ILIKE $1' in query, got: %s", query)
	}

	if len(args) != 1 || args[0] != "%steel%" {
		t.Errorf("expected args = [\"%%steel%%\"], got %v", args)
	}
}

func TestBuildVendorQuery_StatusAndSearchCombined(t *testing.T) {
	query, args := buildVendorQuery(VendorFilters{Status: "Active", Search: "steel"})
	if !strings.Contains(query, "status = $1 AND name ILIKE $2") {
		t.Errorf("expected properly spaced WHERE clause, got: %s", query)
	}
	if len(args) != 2 || args[0] != "Active" || args[1] != "%steel%" {
		t.Errorf("expected args = [\"Active\", \"%%steel%%\"], got %v", args)
	}
}

func TestBuildVendorQuery_ArgOrderMatchesPlaceholderOrder(t *testing.T) {
	query, args := buildVendorQuery(VendorFilters{Status: "Inactive", Search: "metal"})
	idx1 := strings.Index(query, "$1")
	idx2 := strings.Index(query, "$2")
	if idx1 == -1 || idx2 == -1 || idx1 > idx2 {
		t.Fatalf("expected $1 to appear before $2 in query, got: %s", query)
	}
	if args[0] != "Inactive" {
		t.Errorf("expected args[0] to be 'Inactive' (matches $1), got %v", args[0])
	}
	if args[1] != "%metal%" {
		t.Errorf("expected args[1] to be '%%metal%%' (matches $2), got %v", args[1])
	}
}

func TestBuildVendorQuery_AlwaysOrdersByName(t *testing.T) {
	query, _ := buildVendorQuery(VendorFilters{})
	if !strings.HasSuffix(strings.TrimSpace(query), "ORDER BY name") {
		t.Errorf("expected query to end with ORDER BY name, got: %s", query)
	}
}
