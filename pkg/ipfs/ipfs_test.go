package ipfs

import (
	"testing"
)

func TestMockClientRoundTrip(t *testing.T) {
	mock := NewMockClient()

	data := []byte(`{"test": "hello world"}`)
	cid, err := mock.Pin(data)
	if err != nil {
		t.Fatalf("Pin() error = %v", err)
	}
	if cid == "" {
		t.Fatal("Pin() returned empty CID")
	}

	retrieved, err := mock.Retrieve(cid)
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}
	if string(retrieved) != string(data) {
		t.Errorf("Retrieve() = %s, want %s", string(retrieved), string(data))
	}
}

func TestMockClientJSONRoundTrip(t *testing.T) {
	mock := NewMockClient()

	type testStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	original := testStruct{Name: "test", Value: 42}
	cid, err := mock.PinJSON(original)
	if err != nil {
		t.Fatalf("PinJSON() error = %v", err)
	}

	var retrieved testStruct
	if err := mock.RetrieveJSON(cid, &retrieved); err != nil {
		t.Fatalf("RetrieveJSON() error = %v", err)
	}

	if retrieved.Name != original.Name || retrieved.Value != original.Value {
		t.Errorf("RetrieveJSON() = %+v, want %+v", retrieved, original)
	}
}

func TestMockClientNotFound(t *testing.T) {
	mock := NewMockClient()

	_, err := mock.Retrieve("nonexistent-cid")
	if err == nil {
		t.Error("Retrieve() with nonexistent CID should return error")
	}
}
