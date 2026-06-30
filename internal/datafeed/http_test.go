package datafeed

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type MockData struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func TestDoRequest(t *testing.T) {
	// 1. Setup a test server
	expectedData := MockData{ID: 1, Name: "Gopher"}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, r *http.Request) {
		// Verify method
		if r.Method != http.MethodGet {
			t.Errorf("expected GET request, got %s", r.Method)
		}

		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(expectedData)
	}))
	defer server.Close()

	// 2. Execute the function
	ctx := context.Background()
	client := server.Client() // Use the client configured for the test server

	result, err := doRequest[MockData](ctx, client, server.URL)

	// 3. Assertions using testing.T
	if err != nil {
		t.Fatalf("doRequest failed unexpectedly: %v", err)
	}

	if result.ID != expectedData.ID {
		t.Errorf("expected ID %d, got %d", expectedData.ID, result.ID)
	}

	if result.Name != expectedData.Name {
		t.Errorf("expected Name %q, got %q", expectedData.Name, result.Name)
	}
}

func TestDoRequest_ErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	ctx := context.Background()

	_, err := doRequest[MockData](ctx, server.Client(), server.URL)
	if err == nil {
		t.Fatal("expected an error due to 500 status code, but got nil")
	}

	if !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}
}
