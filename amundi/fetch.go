package amundi

import (
	"log"
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/etnz/portfolio"
	"github.com/shopspring/decimal"
)

const amundiPrefix = "Amundi-"

// AmundiID convert an Amundi code (codeFond in their slang) to a portfolio.ID
func AmundiID(code string) portfolio.ID {
	return portfolio.ID(amundiPrefix + code)
}

// AmundiCode return the Amundi code from a portfolio.ID.
func AmundiCode(id portfolio.ID) (code string, ok bool) {
	return strings.CutPrefix(string(id), amundiPrefix)
}

func isAmundi(id portfolio.ID) bool { return strings.HasPrefix(string(id), amundiPrefix) }

// Fetch Amundi Holding reports and create updated prices.
// it fetches reports from today going backwards.
// If eager, it fetches all reports, otherwise it stops as soon as fetched reports stop providing new prices.
func Fetch(headers http.Header, ledger *portfolio.Ledger, eager bool) ([]Change, []portfolio.Transaction, error) {
	products, err := getProducts(headers)
	if err != nil {
		return nil, nil, err
	}

	for _, p := range products {
		log.Printf("Found product %s (%s)", p.Name, p.ID)
	}

	// Extract from the ledger the required information relative to Amundi's asset.
	inceptionDate, knownPrices, securities := amundiInfo(ledger)

	updatePoints, err := fetchAll(headers, products, portfolio.Today(), inceptionDate, knownPrices, eager)
	if err != nil {
		return nil, nil, err
	}
	for k, v := range updatePoints {
		if v.Old != nil {
			log.Println("update", k.ID, k.Date, v)
		} else {
			log.Println("new", k.ID, k.Date, v)
		}
	}

	updates := mergeUpdates(updatePoints, securities)

	// Validate transactions before returning them.
	transactions := make([]portfolio.Transaction, 0, len(updates))
	for _, upd := range updates {
		tx, err := ledger.Validate(upd)
		if err != nil {
			return nil, nil, err
		}
		transactions = append(transactions, tx)
	}

	changeList := slices.Collect(maps.Values(updatePoints))
	slices.SortFunc(changeList, func(a, b Change) int {
		return a.Date.Compare(b.Date)
	})
	slices.SortFunc(transactions, func(a, b portfolio.Transaction) int {
		return a.When().Compare(b.When())
	})

	return changeList, transactions, nil
}

// point is a private struct to hold together the date and ID of a price update.
type point struct {
	Date portfolio.Date
	ID   portfolio.ID
}
type Change struct {
	Date portfolio.Date
	ID   portfolio.ID
	Old  *decimal.Decimal // possibly nil
	New  decimal.Decimal
}

func mergeUpdates(updatePoints map[point]Change, securities map[portfolio.ID][]portfolio.Security) (updates []portfolio.UpdatePrice) {

	// Group together updates by their date
	m := make(map[portfolio.Date]portfolio.UpdatePrice)
	for k, v := range updatePoints {
		// Compute the update per Ticker
		upd := make(map[string]decimal.Decimal)
		for _, sec := range securities[k.ID] {
			upd[sec.Ticker()] = v.New
		}

		// Update the UpdatePrice accordingly
		if up, ok := m[k.Date]; !ok {
			m[k.Date] = portfolio.NewUpdatePrices(k.Date, upd)
		} else {
			maps.Copy(up.Prices, upd) // that should be ok, since up is a copy, but Prices is a pointer.
		}
	}
	return slices.Collect(maps.Values(m))
}

// amundiInfo scan the ledger for info relative to Amundi Assets.
// prices for amundi tickers, and ID->[]tickers mapping.
func amundiInfo(ledger *portfolio.Ledger) (inceptionDate portfolio.Date, known map[point]decimal.Decimal, securities map[portfolio.ID][]portfolio.Security) {
	known = make(map[point]decimal.Decimal)

	// Fill a map of Amundi's tickers.
	tickers := make(map[string]portfolio.Security)
	securities = make(map[portfolio.ID][]portfolio.Security)
	for sec := range ledger.AllSecurities() {
		if _, ok := AmundiCode(sec.ID()); ok {
			tickers[sec.Ticker()] = sec
			securities[sec.ID()] = append(securities[sec.ID()], sec)
		}
	}

	// Scan all transactions to fill the known map, and to compute the Amundi inception date.
	inceptionDate = portfolio.Today() // default to today.
	for _, tx := range ledger.Transactions(portfolio.AcceptAll) {
		switch v := tx.(type) {
		case portfolio.UpdatePrice:
			for ticker, price := range v.PricesIter() {
				if sec, ok := tickers[ticker]; ok {

					known[point{tx.When(), sec.ID()}] = price
				}
			}
		case portfolio.Declare:
			// make inceptionDate the oldest amundi-related declare statement.
			if v.Date.Before(inceptionDate) && isAmundi(v.ID) {
				inceptionDate = v.Date
			}
		}
	}
	return inceptionDate, known, securities
}

func fetchAll(headers http.Header, products []Product, start portfolio.Date, end portfolio.Date, known map[point]decimal.Decimal, eager bool) (updatePoints map[point]Change, err error) {

	// This map will store the new price changes.
	updatePoints = make(map[point]Change)

	// Done flag per product.
	dones := make([]bool, len(products))

	// Iterate from the end date backwards to the start date
	for d := start; !d.Before(end); d = d.Add(-1) {
		// If all product are done, there is no need to continue the loop.
		if !eager {
			allDone := true
			for _, done := range dones {
				allDone = allDone && done
			}
			if allDone {
				break // the fetching loop for all products.
			}
		}

		log.Printf("Fetching holdings as of %s: to get assets prices.", d)
		for i, p := range products {

			// Decide wheter or not it should fetch holdings for this product.
			if dones[i] && !eager {
				continue
			}

			updates, err := getProductHolding(headers, p, d)
			if err != nil {
				// Fail all if there is one error.
				// Otherwise, it would create "gaps" in the data, making the backward strategy
				// invalid.
				return nil, err
			}

			// Check every prices for that product on that day, and mark it as undone, if there was unknown stuff.
			productIsDone := true
			for _, u := range updates {

				// Caveat: updates may contains updates about securities that are not held , and should be ignored.
				if u.Position.Quantity.IsZero() {
					continue
				}

				// TODO: compare the Amundi position with the ledger one to check for differences.

				// New change candidate: pt -> c
				c := Change{
					Date: u.Date,
					ID:   u.ID(),
					New:  u.Value,
				}
				pt := point{Date: u.Date, ID: u.ID()}

				// Check if that price is already known
				if knownPrice, ok := known[pt]; ok {
					if knownPrice.Equal(c.New) {
						// New price is known and identical, no need to update.
						continue
					}
					// New price is known but different, record the old price
					c.Old = &knownPrice
				}
				// Candidate price is valid, adding it. Note it could very well already exist in the map.
				// It is tempty to add it to the known map too, but that would be a mistake, causing problems
				// accross the week-end. Indeed, if we have missing data on thursday and friday, and fetching on Monday.
				// Balance on Monday only contains new data for Friday.
				// Balance on Sunday only contains data for Friday too. If we had added them to the known map,
				// this would stop the algorithm, and we would never get the thursday values.
				updatePoints[pt] = c
				productIsDone = false
			}
			if productIsDone {
				dones[i] = true
			}
		}
	}
	return updatePoints, nil

}
