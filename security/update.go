package security

import (
	"errors"
	"log"

	"github.com/etnz/portfolio/date"
)

// This file contains functions to update the database with latest prices.
// updatable are:
// - security available at EODHD. If the security has currently no data, a default starting date is used.
// - forex pairs.
//
// In order to figure that out, the security.ID is used to identify the security ISIN and MIC where applicable,
// or the forex pair.
// there might be other thypes of securities, but they are not supported by update, yet (like privately traded assets)

func (db *DB) Update() error {

	yesterday := date.Today().Add(-1)

	var errs error

	for ticker, sec := range db.content {
		latest, _ := sec.Prices().Latest()
		if !latest.Before(yesterday) {
			continue
		} // we already have the latest price, so we skip this security.

		origin := date.New(2020, 1, 1) // the default starting date for securities that have no data yet.
		if latest.Before(origin) {
			latest = origin // we use the default starting date.
		}

		// the prices we are going to get from the EODHD API.
		var prices date.History[float64]

		isin, mic, err := sec.ID().MSSI()
		if err == nil {
			// this is an MSSI security that should be available at EODHD.

			prices, err = eodhdDailyISIN(eodhdApiKey(), isin, mic, latest.Add(1), yesterday)
			if err != nil {
				// if we cannot get the prices, we just skip this security.
				// but we log the error.
				errs = errors.Join(errs, err)
				continue
			}
		}
		base, quote, err := sec.ID().CurrencyPair()
		if err == nil {
			// this is a forex pair that should be available at EODHD.
			prices, err = eodhdDailyFrom(eodhdApiKey(), base, quote, latest.Add(1), yesterday)
			if err != nil {
				// if we cannot get the prices, we just skip this security.
				// but we log the error.
				errs = errors.Join(errs, err)
				continue
			}
		}

		// prices now contains the price updates.

		if prices.Len() == 0 {
			log.Printf("no prices found for security %q (%v) between %s and %s", ticker, sec.ID(), latest.Add(1).String(), yesterday.String())
		}
		//append all prices to the security.
		for day, price := range prices.Values() {
			sec.Prices().Append(day, price)
		}
	}
	return errs
}
