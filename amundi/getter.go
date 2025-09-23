package amundi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/etnz/portfolio"
	"github.com/shopspring/decimal"
)

// wget little helper to retrieve payload from http.
func wget(uri string, header http.Header) ([]byte, error) {
	r, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		log.Printf("URI=%s", uri)
		return nil, fmt.Errorf("cannot create http request %q: %w", uri, err)
	}
	r.Header = header

	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		log.Printf("URI=%s, status=%q", uri, resp.Status)
		return nil, fmt.Errorf("cannot execute http request: %w", err)
	}
	body := resp.Body
	defer body.Close()

	// reading in a buffer to be able to print the json in debug mode
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, body); err != nil {
		return nil, fmt.Errorf("cannot read receiving http body: %w", err)
	}

	return buf.Bytes(), nil
}

// Product represent Amundi Accounting product like P.E.E or P.E.R etc.
type Product struct {
	ID     string  `json:"idProduit"`      // The product ID
	Amount float64 `json:"montantBrut"`    // the latest raw amount for the product
	Type   string  `json:"typeIdProduit"`  // there are different types of product with different URI to query
	Name   string  `json:"libelleProduit"` // for debug
}

// getProducts fetchs all products for the logged in user.
func getProducts(headers http.Header) (products []Product, err error) {
	const uriProducts = "https://epargnant.amundi-ee.com/api/individu/produitsEpargne?codeRegroupement=ER%2CRC%2CES"
	// query uriProducts
	// sample of response (I faked the numbers)
	// 	[
	//     {
	//         "idProduit": "1-Xy345sx",
	//         "typeIdProduit": "idDispositif",
	//         "libelleProduit": "Plan Epargne Entreprise",
	//         "typeProduit": "PEE",
	//         "codeRegroupement": "ES",
	//         "idEntreprise": "1-XxXxX",
	//         "nomEntreprise": "XxXxXxX",
	//         "codeEntreprise": "345345",
	//         "montantBrut": 35345345.94,
	//         "montantNet": 6546456.94,
	//         "montantPMV": 3456456.9,
	//         "flagAbonde": false,
	//         "modeGestion": "LIBRE",
	//         "ordreAffichage": 3305,
	//         "montantBrutDivEP": 3456456.94,
	//         "compositionProduit": [
	//             "1-OTGU37"
	//         ],
	//         "flagVv": false
	//     }
	// ]

	data, err := wget(uriProducts, headers)
	if err != nil {
		return nil, fmt.Errorf("error querying products: %w", err)
	}

	if err := json.Unmarshal(data, &products); err != nil {
		return nil, fmt.Errorf("could not decode amundi products' json: %w", err)
	}
	return products, nil
}

// AssetHolding contains an excerpt of the AssetHolding report from Amundi for a particular asset.
type AssetHolding struct {
	AssetID  string          `json:"codeFonds"`    // Amundi's own internal ID (just a number).
	Date     portfolio.Date  `json:"dateVl"`       // Asset's Price date.
	Value    decimal.Decimal `json:"vl"`           // Asset's price.
	Label    string          `json:"libelleFonds"` // Asset name: for debug.
	Position struct {
		Quantity decimal.Decimal `json:"nbParts"`     // Share quantity held.
		Amount   decimal.Decimal `json:"montantBrut"` // total amount held (~ Quantity * Value)
	} `json:"positions"` // Current position in the Holding report.
}

// ID return the porfolio ID for this Update.
func (u AssetHolding) ID() portfolio.ID       { return AmundiID(u.AssetID) }
func (u AssetHolding) Price() decimal.Decimal { return u.Value }

// getProductHolding retrieve product updates for all the assets in this product.
//
// Caveat1: AssetHolding are not necessarily on the day requested.
//
// Caveat2: AssetHolding are also available for funds that are not held.
func getProductHolding(headers http.Header, p Product, day portfolio.Date) (prices []AssetHolding, err error) {
	uris := map[string]string{
		"idDispositif": "https://epargnant.amundi-ee.com/api/individu/produitsEpargne/idDispositif/",
		"affiliation":  "https://epargnant.amundi-ee.com/api/individu/produitsEpargne/affiliation/",
		// the only known right now.
	}
	// regarless the uri for product Type, they all contains the exact same payload:
	var snapshot struct {
		Updates []AssetHolding `json:"fonds"`
	}

	// NOTE: to get position is a bit trickier: "positions" is a sibling of codeFonds
	// "positions": {
	//     "montantBrut": 546345.1,
	//     "nbParts": 29.43758,
	// Caveat: Amundi sometimes returns this:
	//    "montantBrut": 0,
	//    "nbParts": 0.0001,  <----- yes!!
	// For non held positions.

	uri, known := uris[p.Type]
	if !known {
		log.Printf("Product=%#v", p)
		return nil, fmt.Errorf("unknown product type %q", p.Type)
	}
	// and they all have the same arguments
	uri = uri + url.PathEscape(p.ID) + "?date=" + url.QueryEscape(day.Format("2006-01-02T15:04:05Z"))

	data, err := wget(uri, headers)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, &snapshot); err != nil {
		log.Printf("json=```\n\n%s\n\n```", string(data))
		return nil, fmt.Errorf("could not decode amundi snapshot json: %w", err)
	}

	// HOT FIX:
	// Sometimes Amundi API returns:
	//     "montantBrut": 0,
	//     "nbParts": 0.0001,  <----- yes!!
	//
	for i, u := range snapshot.Updates {
		// For safety we'll force both to be 0 if one is.
		if u.Position.Amount.IsZero() || u.Position.Quantity.IsZero() {
			// that is a problem that need fixing
			u.Position.Quantity = decimal.Zero
			u.Position.Amount = decimal.Zero
			snapshot.Updates[i] = u
		}
	}
	return snapshot.Updates, nil

}
