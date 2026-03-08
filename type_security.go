package portfolio

// Security represents a publicly or privately tradeable asset, such as a stock, ETF, or currency pair.
type Security struct {
	id          ID     // The unique, standardized identifier (e.g., MSSI, CurrencyPair).
	ticker      string // The human-friendly ticker used in the portfolio.
	currency    string // The currency in which the security is traded.
	description string // A user-provided description for the security.
	fetcher     string // The URI for fetching market data (e.g. yahoo://MC.PA).
}

func NewSecurity(id ID, ticker, currency, description, fetcher string) Security {
	return Security{
		id:          id,
		ticker:      ticker,
		currency:    currency, // prices are initialized in MarketData
		description: description,
		fetcher:     fetcher,
	}
}

// ID returns the unique, standardized identifier of the security.
func (s Security) ID() ID {
	return s.id
}

// Ticker returns the human-friendly ticker symbol of the security.
func (s Security) Ticker() string {
	return s.ticker
}

func (s Security) Currency() string {
	return s.currency
}

// Description returns the user-provided description for the security.
func (s Security) Description() string {
	return s.description
}

// Fetcher returns the fetch URI for the security.
func (s Security) Fetcher() string {
	return s.fetcher
}
