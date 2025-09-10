package portfolio

var (
	AAPL, _   = NewMSSI("US0378331005", "XNAS")
	GOOG, _   = NewMSSI("US38259P5089", "XNAS")
	USDEUR, _ = NewCurrencyPair("USD", "EUR")
)

// EUR is a helper for test to create euro money from const
func EUR(v float64) Money { return M(v, "EUR") }

// USD is a helper for test to create usd money from const
func USD(v float64) Money { return M(v, "USD") }

// NO is a helper for test to create money from const wit no currency set
func NO(v float64) Money { return M(v, "") }
