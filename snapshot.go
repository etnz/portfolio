package portfolio

import (
	"iter"
)

// Snapshot represents a view of the portfolio at a single point in time.
// It is a stateless calculator that computes all values on-the-fly by
// processing journal events up to its 'on' date.
type Snapshot struct {
	journal *Journal
	on      Date
}

// On returns the date of the snapshot.
func (s *Snapshot) On() Date {
	return s.on
}

// --- private calculation helpers ---

// events returns an iterator over journal events up to the snapshot's date.
func (s *Snapshot) events() iter.Seq[event] {
	return func(yield func(event) bool) {
		for _, e := range s.journal.events {
			if e.date().After(s.on) {
				break
			}
			if !yield(e) {
				return
			}
		}
	}
}

// sum iterates over a sequence of keys (like tickers or currencies),
// applies a metric function to each key to get a Money value, converts
// that value to the reporting currency, and returns the total sum.
func (s *Snapshot) sum(iterator iter.Seq[string], metricFunc func(string) Money) Money {
	total := M(0, s.journal.cur)
	for key := range iterator {
		value := metricFunc(key)
		// The metric function returns a value in the entity's currency.
		// We convert it to the reporting currency before summing.
		total = total.Add(s.Convert(value))
	}
	return total
}

// VirtualAssetValue simulates the growth of a 1-unit investment in a security.
// This is the core of the Time-Weighted Return calculation. It tracks a "virtual"
// portfolio that buys into a security when the actual position goes from zero to
// non-zero, and sells out when the actual position returns to zero.
func (s *Snapshot) VirtualAssetValue(ticker string) Money {
	// This simulates a virtual portfolio for the given ticker.
	// It starts with 1 unit of the security's currency.
	virtualCash := M(1, "")
	var virtualPosition Quantity // The number of virtual shares held.

	// These track the state of the *actual* portfolio.
	var actualPosition Quantity
	var lastPrice Money
	for e := range s.events() {
		switch v := e.(type) {
		case declareSecurity:
			if v.ticker == ticker {
				lastPrice = M(0, v.currency)
				virtualCash = M(1, v.currency)
			}
		case acquireLot:
			if v.security == ticker {
				if actualPosition.IsZero() {
					// First buy. The virtual portfolio uses its cash to buy virtual shares.
					lastPrice = v.cost.Div(v.quantity)
					virtualPosition = virtualCash.Mul(v.quantity).DivPrice(v.cost)
					virtualCash = M(0, virtualCash.Currency()) // All cash is now invested.
				}
				actualPosition = actualPosition.Add(v.quantity)
			}
		case disposeLot:
			if v.security == ticker {
				actualPosition = actualPosition.Sub(v.quantity)
				if actualPosition.IsZero() {
					// Actual position is down to zero. "Sell all" the virtual position.
					virtualCash = lastPrice.Mul(virtualPosition)
					virtualPosition = Q(0)
				}
			}
		case updatePrice:
			if v.security == ticker {
				lastPrice = v.price
			}
		case splitShare:
			if v.security == ticker {
				num, den := Q(v.numerator), Q(v.denominator)
				actualPosition = actualPosition.Mul(num).Div(den)
				// The virtual position also splits.
				virtualPosition = virtualPosition.Mul(num).Div(den)
			}
		}
	}

	// Calculate the final value of the virtual portfolio.
	return virtualCash.Add(lastPrice.Mul(virtualPosition))
}

// VirtualTotalValue simulates the growth of a 1-unit investment in the entire portfolio.
// This is the core of the portfolio-wide Time-Weighted Return calculation. It tracks
// the portfolio's value and adjusts for external cash flows.
func (s *Snapshot) VirtualTotalValue() Money {
	return s.TotalPortfolio().Sub(s.TotalCashFlow())
}

// Price finds the last known price for a security on or before the snapshot's date.
func (s *Snapshot) Price(ticker string) Money {
	var lastPrice Money
	if sec, ok := s.SecurityDetails(ticker); !ok {
		return Money{} // undeclared security, cannot have price
	} else {
		lastPrice = M(0, sec.Currency())
	}
	for e := range s.events() {
		if u, ok := e.(updatePrice); ok && u.security == ticker {
			lastPrice = u.price
		}
	}
	return lastPrice
}

// --- public calculation helpers ---

// Position calculates the quantity held of a single security on the snapshot's date.
func (s *Snapshot) Position(ticker string) Quantity {
	var position Quantity
	for e := range s.events() {
		switch v := e.(type) {
		case acquireLot:
			if v.security == ticker {
				position = position.Add(v.quantity)
			}
		case disposeLot:
			if v.security == ticker {
				position = position.Sub(v.quantity)
			}
		case splitShare:
			if v.security == ticker {
				num, den := Q(v.numerator), Q(v.denominator)
				position = position.Mul(num).Div(den)
			}
		}
	}
	return position
}

// SecurityDetails finds the declaration for a given ticker.
func (s *Snapshot) SecurityDetails(ticker string) (Security, bool) {
	for e := range s.events() {
		if d, ok := e.(declareSecurity); ok && d.ticker == ticker {
			return NewSecurity(d.id, d.ticker, d.currency), true
		}
	}
	return Security{}, false
}

// MarketValue calculates the market value of a single security on the snapshot's date.
func (s *Snapshot) MarketValue(ticker string) Money {
	pos := s.Position(ticker)
	price := s.Price(ticker)
	return price.Mul(pos)
}

// Cash returns the balance of a specific cash account on the snapshot's date.
func (s *Snapshot) Cash(currency string) Money {
	balance := M(0, currency)
	for e := range s.events() {
		switch v := e.(type) {
		case creditCash:
			if v.currency() == currency {
				balance = balance.Add(v.amount)
			}
		case debitCash:
			if v.currency() == currency {
				balance = balance.Sub(v.amount)
			}
		}
	}
	return balance
}

// Dividends calculates the total income received from
// dividends for a specific security since inception.
func (s *Snapshot) Dividends(ticker string) Money {
	var totalDividends Money
	var position Quantity

	for e := range s.events() {
		switch v := e.(type) {
		case acquireLot:
			if v.security == ticker {
				position = position.Add(v.quantity)
			}
		case splitShare:
			if v.security == ticker {
				num, den := Q(v.numerator), Q(v.denominator)
				position = position.Mul(num).Div(den)
			}
		case disposeLot:
			if v.security == ticker {
				position = position.Sub(v.quantity)
			}
		case receiveDividend:
			if v.security == ticker {
				totalAmount := v.amount.Mul(position)
				totalDividends = totalDividends.Add(totalAmount)
			}
		}
	}
	return totalDividends
}

// CostBasis calculates the total cost basis of a security held on the snapshot's date.
func (s *Snapshot) CostBasis(ticker string, method CostBasisMethod) Money {
	switch method {
	case AverageCost:
		var totalQuantity Quantity
		var totalCost Money
		for e := range s.events() {
			switch v := e.(type) {
			case acquireLot:
				if v.security == ticker {
					totalQuantity = totalQuantity.Add(v.quantity)
					totalCost = totalCost.Add(v.cost)
				}
			case splitShare:
				if v.security == ticker {
					num, den := Q(v.numerator), Q(v.denominator)
					totalQuantity = totalQuantity.Mul(num).Div(den)
				}
			case disposeLot:
				if v.security == ticker {
					if !totalQuantity.IsZero() {
						costOfSale := totalCost.Mul(v.quantity).Div(totalQuantity)
						totalCost = totalCost.Sub(costOfSale)
					}
					totalQuantity = totalQuantity.Sub(v.quantity)
				}
			}
		}
		return totalCost
	case FIFO:
		var securityLots lots
		for e := range s.events() {
			switch v := e.(type) {
			case acquireLot:
				if v.security == ticker {
					newLot := lot{Date: v.on, Quantity: v.quantity, Cost: v.cost}
					securityLots = append(securityLots, newLot)
				}
			case splitShare:
				if v.security == ticker {
					num, den := Q(v.numerator), Q(v.denominator)
					// we need to split shares in all lots
					for i := range securityLots {
						securityLots[i].Quantity = securityLots[i].Quantity.Mul(num).Div(den)
					}
				}
			case disposeLot:
				if v.security == ticker {
					securityLots = securityLots.sell(v.quantity)
				}
			}
		}
		var totalCost Money
		for _, l := range securityLots {
			totalCost = totalCost.Add(l.Cost)
		}
		return totalCost
	default:
		return Money{} // Or handle error
	}
}

// RealizedGains calculates the sum of all profits and losses 'locked in'
// through the sale of a specific security since inception.
func (s *Snapshot) RealizedGains(ticker string, method CostBasisMethod) Money {
	switch method {
	case AverageCost:
		var realizedGain Money
		var totalQuantity Quantity
		var totalCost Money
		for e := range s.events() {
			switch v := e.(type) {
			case acquireLot:
				if v.security == ticker {
					totalQuantity = totalQuantity.Add(v.quantity)
					totalCost = totalCost.Add(v.cost)
				}
			case splitShare:
				if v.security == ticker {
					num, den := Q(v.numerator), Q(v.denominator)
					totalQuantity = totalQuantity.Mul(num).Div(den)
				}
			case disposeLot:
				if v.security == ticker {
					costOfSale := totalCost.Mul(v.quantity).Div(totalQuantity)
					gain := v.proceeds.Sub(costOfSale)
					realizedGain = realizedGain.Add(gain)
					totalCost = totalCost.Sub(costOfSale)
					totalQuantity = totalQuantity.Sub(v.quantity)
				}
			}
		}
		return realizedGain
	case FIFO:
		var realizedGain Money
		var securityLots lots
		for e := range s.events() {
			switch v := e.(type) {
			case acquireLot:
				if v.security == ticker {
					newLot := lot{Date: v.on, Quantity: v.quantity, Cost: v.cost}
					securityLots = append(securityLots, newLot)
				}
			case splitShare:
				if v.security == ticker {
					num, den := Q(v.numerator), Q(v.denominator)
					// we need to split shares in all lots
					for i := range securityLots {
						securityLots[i].Quantity = securityLots[i].Quantity.Mul(num).Div(den)
					}
				}
			case disposeLot:
				if v.security == ticker {
					costOfSale := securityLots.fifoCostOfSelling(v.quantity)
					gain := v.proceeds.Sub(costOfSale)
					realizedGain = realizedGain.Add(gain)
					securityLots = securityLots.sell(v.quantity)
				}
			}
		}
		return realizedGain
	default:
		return Money{} // Or handle error
	}
}

// NetTradingFlow calculates the total net cash invested into or divested from
// a specific security since inception. A positive value indicates a net cash
// outflow (more spent on buys than received from sells).
func (s *Snapshot) NetTradingFlow(ticker string) Money {
	var netFlow Money
	for e := range s.events() {
		switch v := e.(type) {
		case acquireLot:
			if v.security == ticker {
				netFlow = netFlow.Add(v.cost)
			}
		case disposeLot:
			if v.security == ticker {
				netFlow = netFlow.Sub(v.proceeds)
			}
		}
	}
	return netFlow
}

// CashFlow calculates the total net cash that has moved into or out
// of the portfolio from external sources for a specific currency since inception.
func (s *Snapshot) CashFlow(currency string) Money {
	flow := M(0, currency)
	for e := range s.events() {
		switch v := e.(type) {
		case creditCash:
			if v.external && v.currency() == currency {
				flow = flow.Add(v.amount)
			}
		case debitCash:
			if v.external && v.currency() == currency {
				flow = flow.Sub(v.amount)
			}
		}
	}
	return flow
}

// Counterparty returns the balance of a specific counterparty account on the snapshot's date.
func (s *Snapshot) Counterparty(account string) Money {
	var balance Money
	for e := range s.events() {
		switch v := e.(type) {
		case creditCounterparty:
			if v.account == account {
				balance = balance.Add(v.amount)
			}
		case debitCounterparty:
			if v.account == account {
				balance = balance.Sub(v.amount)
			}
		case declareCounterparty:
			// This ensures the money has the correct currency even if balance is zero.
			if v.account == account && balance.IsZero() {
				balance = M(0, v.currency)
			}
		}
	}
	return balance
}

// Securities returns an iterator over all declared security tickers up to the snapshot's date.
// The order is based on the date of their declaration.
func (s *Snapshot) Securities() iter.Seq[string] {
	return func(yield func(string) bool) {
		seen := make(map[string]struct{})
		for e := range s.events() {
			if v, ok := e.(declareSecurity); ok {
				if _, exists := seen[v.ticker]; !exists {
					seen[v.ticker] = struct{}{}
					if !yield(v.ticker) {
						return
					}
				}
			}
		}
	}
}

// Counterparties returns an iterator over all declared counterparty accounts up to the snapshot's date.
// The order is based on the date of their declaration.
func (s *Snapshot) Counterparties() iter.Seq[string] {
	return func(yield func(string) bool) {
		seen := make(map[string]struct{})
		for e := range s.events() {
			if v, ok := e.(declareCounterparty); ok {
				if _, exists := seen[v.account]; !exists {
					seen[v.account] = struct{}{}
					if !yield(v.account) {
						return
					}
				}
			}
		}
	}
}

// Currencies returns an iterator over all currencies encountered up to the snapshot's date.
// This includes the reporting currency, currencies from cash transactions, and security/counterparty declarations.
// The order is based on their first appearance.
func (s *Snapshot) Currencies() iter.Seq[string] {
	return func(yield func(string) bool) {
		seen := make(map[string]struct{})

		// Helper to yield a currency if it's new
		process := func(cur string) bool {
			if cur != "" {
				if _, exists := seen[cur]; !exists {
					seen[cur] = struct{}{}
					if !yield(cur) {
						return false // Stop iteration
					}
				}
			}
			return true // Continue iteration
		}

		// Ensure reporting currency is always included first
		if !process(s.journal.cur) {
			return
		}

		for e := range s.events() {
			var currency string
			switch v := e.(type) {
			case creditCash:
				currency = v.currency()
			case debitCash:
				currency = v.currency()
			case declareSecurity:
				currency = v.currency
			case declareCounterparty:
				currency = v.currency
			}

			if !process(currency) {
				return
			}
		}
	}
}

// Convert converts a monetary amount into the default currency.
func (s *Snapshot) Convert(amount Money) Money {
	rate := s.ExchangeRate(amount.Currency())
	// The rate is how much 1 unit of the foreign currency is worth in the reporting currency.
	// So, we multiply the foreign amount's value by the rate.
	return rate.Mul(Q(amount.value))
}

// ExchangeRate finds the last known exchange rate for a given currency on or before the snapshot's date.
// The rate is the value of 1 unit of the foreign currency in the portfolio's reporting currency.
func (s *Snapshot) ExchangeRate(currency string) Money {
	if currency == s.journal.cur {
		return M(1, s.journal.cur)
	}
	var lastRate Money
	for e := range s.events() {
		if u, ok := e.(updateForex); ok && u.currency == currency {
			lastRate = u.rate
		}
	}
	if lastRate.IsZero() {
		return M(0, s.journal.cur)
	}
	return lastRate
}

// TotalMarket returns the total market value of all securities in the portfolio.
func (s *Snapshot) TotalMarket() Money {
	return s.sum(s.Securities(), s.MarketValue)
}

// TotalCash returns the total cash balance across all currencies, converted to the reporting currency.
func (s *Snapshot) TotalCash() Money {
	return s.sum(s.Currencies(), s.Cash)
}

// TotalCounterparty returns the total balance across all counterparty accounts, converted to the reporting currency.
func (s *Snapshot) TotalCounterparty() Money {
	return s.sum(s.Counterparties(), s.Counterparty)
}

// TotalPortfolio returns the total value of the portfolio, including securities, cash, and counterparty accounts.
func (s *Snapshot) TotalPortfolio() Money {
	return s.TotalMarket().
		Add(s.TotalCash()).
		Add(s.TotalCounterparty())
}

// UnrealizedGains calculates the paper profit or loss on a security.
// It's the difference between the current market value and the cost basis.
func (s *Snapshot) UnrealizedGains(ticker string, method CostBasisMethod) Money {
	marketValue := s.MarketValue(ticker)
	costBasis := s.CostBasis(ticker, method)
	return marketValue.Sub(costBasis)
}

// TotalCashFlow returns the total cash flow across all currencies, converted to the reporting currency.
func (s *Snapshot) TotalCashFlow() Money {
	return s.sum(s.Currencies(), s.CashFlow)
}

// TotalNetTradingFlow returns the total net trading flow across all securities, converted to the reporting currency.
func (s *Snapshot) TotalNetTradingFlow() Money {
	return s.sum(s.Securities(), s.NetTradingFlow)
}

// TotalRealizedGains returns the total realized gains across all securities, converted to the reporting currency.
func (s *Snapshot) TotalRealizedGains(method CostBasisMethod) Money {
	return s.sum(s.Securities(), func(ticker string) Money {
		return s.RealizedGains(ticker, method)
	})
}

// TotalDividends returns the total dividends received across all securities, converted to the reporting currency.
func (s *Snapshot) TotalDividends() Money {
	return s.sum(s.Securities(), s.Dividends)
}

// TotalUnrealizedGains calculates the total unrealized gains across all securities.
func (s *Snapshot) TotalUnrealizedGains(method CostBasisMethod) Money {
	return s.sum(s.Securities(), func(ticker string) Money {
		return s.UnrealizedGains(ticker, method)
	})
}

// TotalCostBasis calculates the total cost basis of all securities.
func (s *Snapshot) TotalCostBasis(method CostBasisMethod) Money {
	return s.sum(s.Securities(), func(ticker string) Money {
		return s.CostBasis(ticker, method)
	})
}
