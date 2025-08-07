package portfolio

import "testing"

func TestValidateISIN(t *testing.T) {
	testCases := []struct {
		name      string
		isin      string
		expectErr bool
	}{
		{"Valid Apple ISIN", "US0378331005", false},
		{"Valid VW ISIN", "DE0007664039", false},
		{"Invalid Check Digit", "US0378331006", true},
		{"Invalid Length (Short)", "US123", true},
		{"Invalid Length (Long)", "US03783310055", true},
		{"Invalid Format (Contains 'X')", "US037833100X", true},
		{"Invalid Format (lowercase)", "us0378331005", true},
		{"Empty String", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateISIN(tc.isin)
			hasErr := err != nil

			if hasErr != tc.expectErr {
				t.Errorf("ValidateISIN(%q) returned error: %v, want error: %v", tc.isin, err, tc.expectErr)
			}
		})
	}
}

func TestValidateMIC(t *testing.T) {
	testCases := []struct {
		name      string
		mic       string
		expectErr bool
	}{
		{"Valid Nasdaq MIC", "XNAS", false},
		{"Valid Xetra MIC", "XETR", false},
		{"Valid Alphanumeric MIC", "A0B1", false},
		{"Invalid Length (Short)", "XN", true},
		{"Invalid Length (Long)", "XETRA", true},
		{"Invalid Character", "XET-", true},
		{"Invalid Case (lowercase)", "xnas", true},
		{"Empty String", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateMIC(tc.mic)
			hasErr := err != nil

			if hasErr != tc.expectErr {
				t.Errorf("ValidateMIC(%q) returned error: %v, want error: %v", tc.mic, err, tc.expectErr)
			}
		})
	}
}

func TestValidateCurrency(t *testing.T) {
	testCases := []struct {
		name      string
		code      string
		expectErr bool
	}{
		{"Valid USD", "USD", false},
		{"Valid EUR", "EUR", false},
		{"Invalid Length (Short)", "US", true},
		{"Invalid Length (Long)", "USDE", true},
		{"Invalid Character (number)", "US1", true},
		{"Invalid Case (lowercase)", "usd", true},
		{"Empty String", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCurrency(tc.code)
			hasErr := err != nil

			if hasErr != tc.expectErr {
				t.Errorf("ValidateCurrency(%q) returned error: %v, want error: %v", tc.code, err, tc.expectErr)
			}
		})
	}
}

func TestMSSI(t *testing.T) {
	testCases := []struct {
		name       string
		mssi       ID
		expectISIN string
		expectMIC  string
		expectErr  bool
	}{
		{
			name:       "Valid MSSI",
			mssi:       "US0378331005.XNAS",
			expectISIN: "US0378331005",
			expectMIC:  "XNAS",
			expectErr:  false,
		},
		{
			name:      "Invalid Structure (No Separator)",
			mssi:      "US0378331005XNAS",
			expectErr: true,
		},
		{
			name:      "Invalid ISIN Part (Bad Check Digit)",
			mssi:      "US0378331006.XNAS",
			expectErr: true,
		},
		{
			name:      "Invalid ISIN Part (Bad Length)",
			mssi:      "US037833100.XNAS",
			expectErr: true,
		},
		{
			name:      "Invalid MIC Part (Bad Length)",
			mssi:      "DE0007664039.XET",
			expectErr: true,
		},
		{
			name:      "Invalid MIC Part (Bad Format)",
			mssi:      "DE0007664039.xetr",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isin, mic, err := tc.mssi.MSSI()

			hasErr := err != nil
			if hasErr != tc.expectErr {
				t.Errorf("ParseMSSI(%q) unexpected error state. Got error: %v, want error: %v", tc.mssi, err, tc.expectErr)
			}

			// If we expect success, also check the returned values
			if !tc.expectErr {
				if isin != tc.expectISIN {
					t.Errorf("ParseMSSI(%q) incorrect ISIN. Got: %q, want: %q", tc.mssi, isin, tc.expectISIN)
				}
				if mic != tc.expectMIC {
					t.Errorf("ParseMSSI(%q) incorrect MIC. Got: %q, want: %q", tc.mssi, mic, tc.expectMIC)
				}
			}
		})
	}
}

func TestCurrencyPair(t *testing.T) {
	testCases := []struct {
		name        string
		id          ID
		expectBase  string
		expectQuote string
		expectErr   bool
	}{
		{"Valid Pair", "USDJPY", "USD", "JPY", false},
		{"Another Valid Pair", "EURGBP", "EUR", "GBP", false},
		{"Invalid Length (short)", "EUR", "", "", true},
		{"Invalid Length (long)", "EURUSDD", "", "", true},
		{"Invalid Chars (lowercase)", "eurusd", "", "", true},
		{"Invalid Chars (hyphen)", "EUR-USD", "", "", true},
		{"Empty String", "", "", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			base, quote, err := tc.id.CurrencyPair()

			if (err != nil) != tc.expectErr {
				t.Fatalf("ParseCurrencyPair(%q) unexpected error state. Got error: %v, want error: %v", tc.id, err, tc.expectErr)
			}

			// If we expect success, also check the returned values
			if !tc.expectErr {
				if base != tc.expectBase {
					t.Errorf("ParseCurrencyPair(%q) incorrect base currency. Got: %q, want: %q", tc.id, base, tc.expectBase)
				}
				if quote != tc.expectQuote {
					t.Errorf("ParseCurrencyPair(%q) incorrect quote currency. Got: %q, want: %q", tc.id, quote, tc.expectQuote)
				}
			}
		})
	}
}

// TestNewPrivate is a data-driven test for the NewPrivate function.
//
// The rationale for this test is to verify that the parser correctly accepts IDs
// that are at least 7 characters long and contain valid characters, while rejecting
// strings that are too short, contain invalid characters, or resemble other identifier formats (MSSI, CurrencyPair).
func TestNewPrivate(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		expectErr bool
	}{
		// Valid IDs
		{"Valid 7-char ID", "myID123", false},
		{"Valid 7-char ID with Space", "abc 123", false},
		{"Valid Long ID", "a long id 123", false},
		{"Valid (contains char)", "my-ID-123", false},

		// Invalid IDs
		{"Invalid (is a CurrencyPair - fails on length)", "EURUSD", true},
		{"Invalid (resembles MSSI)", "ABCD.EFG", true},
		{"Invalid (too short)", "id", true},
		{"Empty String", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewPrivate(tc.input)
			hasErr := err != nil

			if hasErr != tc.expectErr {
				t.Fatalf("ParseID(%q) unexpected error state. Got error: %v, want error: %v", tc.input, err, tc.expectErr)
			}
		})
	}
}

func TestParseID(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		expectID  ID
		expectErr bool
	}{
		// Valid cases
		{"Valid MSSI", "US0378331005.XNAS", "US0378331005.XNAS", false},
		{"Valid CurrencyPair", "EURUSD", "EURUSD", false},
		{"Valid Private ID", "My Private Fund", "My Private Fund", false},

		// Invalid cases
		{
			name:      "Invalid (Too Short)",
			input:     "short",
			expectErr: true,
		},
		{
			name:      "Invalid (Resembles MSSI but is invalid)",
			input:     "NOTANISIN.MIC",
			expectErr: true,
		},
		{
			name:      "Invalid (Resembles CurrencyPair but is invalid)",
			input:     "eurusd",
			expectErr: true,
		},
		{
			name:      "Empty String",
			input:     "",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			id, err := ParseID(tc.input)
			if (err != nil) != tc.expectErr {
				t.Fatalf("ParseID(%q) unexpected error state. Got error: %v, want error: %v", tc.input, err, tc.expectErr)
			}
			if !tc.expectErr && id != tc.expectID {
				t.Errorf("ParseID(%q) incorrect ID. Got: %q, want: %q", tc.input, id, tc.expectID)
			}
		})
	}
}

func TestMarketData_Prices_NonExistentID(t *testing.T) {
	m := NewMarketData()
	id := ID("NONEXISTENT")

	// Verify that the Prices() iterator does not panic on missing security.
	c := 0
	for range m.Prices(id) {
		c++
	}

	if c != 0 {
		t.Errorf("Prices() for a non-existent ID should return a nil iterator, but it did not")
	}
}
