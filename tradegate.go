package portfolio

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/PaesslerAG/jsonpath"
)

/*
	{
	    "info": {
	        "isin": "LS000IUSD016",
	        "chartType": "mini",
	        "textMaxValue": "high",
	        "textMinValue": "low",
	        "plotlines": [
	            {
	                "label": "previous 1.049",
	                "value": 1.04875,
	                "align": "left",
	                "y": 8,
	                "id": "previousDay",
	                "color": "#333"
	            }
	        ]
	    },
*/
func tradegateLatestEURperUSD() (float64, error) {
	// this is not tradegate ;-)
	addr := "https://www.ls-tc.de/_rpc/json/instrument/chart/dataForInstrument?instrumentId=349938&series=intraday&type=mini"
	var jobj any
	err := jwget(new(http.Client), addr, &jobj)
	if err != nil {
		return math.NaN(), fmt.Errorf("error in wget %q: %w", "EUR/USD", err)
	}
	path := "$.series.intraday.data[-1:][1]"
	jval, err := jsonpath.Get(path, jobj)
	if err != nil {
		return math.NaN(), fmt.Errorf("error parsing %q: %q %w", "EUR/USD", path, err)
	}
	// because jsonpath is never clear about wheter it returns a list of 1 answer, or a single answer:
	// by this call I keep the first one if any
	if jlist, ok := jval.([]any); ok && len(jlist) > 0 {
		jval = jlist[0]
	}

	val, ok := jval.(float64)
	if !ok {
		return math.NaN(), fmt.Errorf("error parsing %q: %q %s %v", "EUR/USD", path, "not a float", jval)
	}
	return val, nil
}

// tradegateLatest update all stocks from latest value exchanged in TrageGate.
// They are all in Eur, so they are converted back to their currency if there is
// a currency attribute in the metadata
func tradegateLatest(name, isin string) (float64, error) {

	base := "https://www.tradegate.de/refresh.php?isin="
	addr := base + isin

	var jobj map[string]any

	err := jwget(new(http.Client), addr, &jobj)
	if err != nil {
		return math.NaN(), fmt.Errorf("error retrieving %q: %w", name, err)
	}
	// last is the last transaction, moves slower than the bid, but the bid can be 0.
	jval := jobj["last"] // or bid
	if s, ok := jval.(string); ok {
		if s == "./." {
			// trade gate show's empty last this way, use the bid instead
			log.Println("'last' is empty, falling back to 'bid'")
			jval = jobj["bid"]
		}
	}
	val, ok := jval.(float64)
	if !ok {
		// sometimes, this weird API returns the value as a string
		sval, ok := jval.(string)
		if !ok {
			return math.NaN(), fmt.Errorf("cannot read value from %q: doesn't have a value and neither a float or string", name)
		}
		//log.Printf("warning: read as string %q", sval)
		sval = strings.ReplaceAll(sval, ",", ".")
		sval = strings.ReplaceAll(sval, " ", "")
		val, err = strconv.ParseFloat(sval, 64)
		if err != nil {
			return math.NaN(), fmt.Errorf("cannot read value from %q: value is an invalid string %q: %w", name, sval, err)
		}
	}
	if val == 0 {
		// sometimes the bid is empty and returns 0
		return math.NaN(), fmt.Errorf("empty bid for %s no value to return: bidsize=%v", name, jobj["bidsize"])
	}
	return val, nil
}
