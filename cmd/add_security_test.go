package cmd

import (
	"testing"
)

func TestGenerateAddCommand(t *testing.T) {
	cmd := NewAddSecurityCmd()

	ticker := "AAPL"
	id := "US0378331005.XETR"
	currency := "EUR"

	expected := "pcs add-security -s='AAPL' -id='US0378331005.XETR' -c='EUR'"
	actual := cmd.GenerateAddCommand(ticker, id, currency)

	if actual != expected {
		t.Errorf("GenerateAddCommand(%q, %q, %q) = %q; want %q", ticker, id, currency, actual, expected)
	}
}