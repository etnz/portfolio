package portfolio

// SplitInfo holds the details of a stock split.
type SplitInfo struct {
	Numerator   int64
	Denominator int64
}

// DividendInfo holds the details of a dividend payment.
type DividendInfo struct {
	Amount float64 // Amount per share
}

// ProviderResponse holds the data returned by a data provider for a single security.
// It's a generic container for prices, splits, and dividends.
type ProviderResponse struct {
	Prices    map[Date]float64
	Splits    map[Date]SplitInfo
	Dividends map[Date]DividendInfo
}
