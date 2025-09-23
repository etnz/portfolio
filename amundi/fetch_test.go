package amundi

import (
	"testing"

	"github.com/etnz/portfolio"
)

func TestAmundiID(t *testing.T) {
	code := "ABC-123"
	expectedID := portfolio.ID("Amundi-ABC-123")
	if got := AmundiID(code); got != expectedID {
		t.Errorf("AmundiID() = %v, want %v", got, expectedID)
	}
}

func TestAmundiCode(t *testing.T) {
	tests := []struct {
		name         string
		id           portfolio.ID
		expectedCode string
		expectedOk   bool
	}{
		{
			name:         "Valid Amundi ID",
			id:           portfolio.ID("Amundi-12345"),
			expectedCode: "12345",
			expectedOk:   true,
		},
		{
			name:         "ID with different prefix",
			id:           portfolio.ID("Other-12345"),
			expectedCode: "Other-12345",
			expectedOk:   false,
		},
		{
			name:         "ID without any prefix",
			id:           portfolio.ID("12345"),
			expectedCode: "12345",
			expectedOk:   false,
		},
		{
			name:         "Empty ID",
			id:           portfolio.ID(""),
			expectedCode: "",
			expectedOk:   false,
		},
		{
			name:         "ID is exactly the prefix",
			id:           portfolio.ID("Amundi-"),
			expectedCode: "",
			expectedOk:   true,
		},
		{
			name:         "ID contains prefix but not at start",
			id:           portfolio.ID("some-Amundi-123"),
			expectedCode: "some-Amundi-123",
			expectedOk:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, ok := AmundiCode(tt.id)
			if code != tt.expectedCode {
				t.Errorf("AmundiCode() code = %v, want %v", code, tt.expectedCode)
			}
			if ok != tt.expectedOk {
				t.Errorf("AmundiCode() ok = %v, want %v", ok, tt.expectedOk)
			}
		})
	}
}
