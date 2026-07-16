package store

import "testing"

func TestParseImportSingle(t *testing.T) {
	raw := []byte(`{"access_token":"tok-1","refresh_token":"r1","email":"a@b.c"}`)
	items, err := ParseImport(raw, "x")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].AccessToken != "tok-1" {
		t.Fatalf("%+v", items)
	}
	if items[0].Label == "" {
		t.Fatal("label")
	}
}

func TestParseImportCredentialsArray(t *testing.T) {
	raw := []byte(`{"credentials":[{"accessToken":"a","name":"one"},{"access_token":"b","label":"two"}]}`)
	items, err := ParseImport(raw, "imp")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("len=%d", len(items))
	}
}

func TestParseImportAccountsMap(t *testing.T) {
	raw := []byte(`{"accounts":{"acc1":{"accessToken":"z","email":"e@x.com"}}}`)
	items, err := ParseImport(raw, "imp")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].AccessToken != "z" {
		t.Fatalf("%+v", items)
	}
}
