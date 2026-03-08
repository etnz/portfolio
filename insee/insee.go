package insee

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/etnz/portfolio"
	"github.com/shopspring/decimal"
)

// Fetch retrieves market data from INSEE for the requested securities and date ranges.
func Fetch(ctx context.Context, security portfolio.Security, from, to portfolio.Date) ([]portfolio.Transaction, error) {
	u, err := url.Parse(security.Fetcher())
	if err != nil {
		return nil, fmt.Errorf("invalid fetcher URI: %w", err)
	}
	idBank := u.Host
	if idBank == "" {
		idBank = u.Path
	}

	series, err := getSeries(ctx, idBank, from, to)
	if err != nil {
		return nil, err
	}

	var updates []portfolio.Transaction
	for date, price := range series.Values {
		if date.Before(from) || date.After(to) {
			continue
		}
		tx := portfolio.NewUpdatePrice(date, security.Ticker(), portfolio.M(decimal.NewFromFloat(price), security.Currency()))
		updates = append(updates, tx)
	}
	return updates, nil
}

// getSeries constructs the URL, downloads, and parses an INSEE time series.
func getSeries(ctx context.Context, idBank string, from, to portfolio.Date) (*Series, error) {
	// INSEE date is relative to the data, while we are sending date about when the data was publish, INSEE index can take up to 2 months to be published.
	from = from.AddMonth(-3)
	startYear, startPeriod := from.Year(), from.Quarter()
	endYear, endPeriod := to.Year(), to.Quarter()

	url := fmt.Sprintf("https://bdm.insee.fr/series/%s/csv?lang=fr&ordre=antechronologique&periodeDebut=%d&anneeDebut=%d&periodeFin=%d&anneeFin=%d&revision=sansrevisions",
		idBank,
		startPeriod, startYear,
		endPeriod, endYear,
	)
	log.Println("Downloading from INSEE:", url)

	var resp *http.Response
	var err error
	const retries = 3
	for i := 1; i <= retries; i++ {
		var e error
		req, reqErr := http.NewRequestWithContext(ctx, "GET", url, nil)
		if reqErr != nil {
			return nil, reqErr
		}
		resp, e = http.DefaultClient.Do(req)
		if e != nil {
			log.Println("Error downloading from INSEE Retrying.", i)
			err = errors.Join(err, e)
			continue
		}
		break
	}
	if err != nil {
		return nil, fmt.Errorf("failed to download from INSEE for ID %s after %d retries: %w", idBank, retries, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download from INSEE for ID %s: received status %s", idBank, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to open zip archive from INSEE response: %w", err)
	}

	var foundFiles []string
	for _, f := range zipReader.File {
		filename := f.Name
		foundFiles = append(foundFiles, filename)
		if filename == "valeurs_trimestrielles.csv" || filename == "valeurs_mensuelles.csv" {
			log.Println("Found", filename)
			csvFile, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open '%s' from zip archive: %w", filename, err)
			}
			defer csvFile.Close()
			return parseSeries(csvFile)
		}
	}

	return nil, fmt.Errorf("could not find a values file (mensuelles or trimestrielles) in downloaded zip file for ID %s (found: %s)", idBank, strings.Join(foundFiles, ", "))
}

// Series holds the data from an INSEE time series CSV file.
type Series struct {
	Libelle    string
	IDBank     string
	LastUpdate time.Time
	Values     map[portfolio.Date]float64
}

// parseInseeDate parses a string like "2025-T2" or "2025-08" into a portfolio.Date
// representing the end of that period.
func parseInseeDate(s string) (portfolio.Date, error) {
	// Try quarterly format: "YYYY-TQ"
	if strings.Contains(s, "-T") {
		return parseQuarterlyDate(s)
	}

	// Try monthly format: "YYYY-MM"
	parts := strings.Split(s, "-")
	if len(parts) == 2 {
		year, err := strconv.Atoi(parts[0])
		if err != nil {
			return portfolio.Date{}, fmt.Errorf("invalid year in monthly date %q: %w", s, err)
		}
		month, err := strconv.Atoi(parts[1])
		if err != nil || month < 1 || month > 12 {
			return portfolio.Date{}, fmt.Errorf("invalid month in monthly date %q: %w", s, err)
		}
		return portfolio.NewDate(year, time.Month(month)+1, 0), nil
	}
	return portfolio.Date{}, fmt.Errorf("unrecognized insee date format: %q", s)
}

// parseQuarterlyDate parses a string like "2025-T2" into a portfolio.Date
// representing the end of that quarter.
func parseQuarterlyDate(s string) (portfolio.Date, error) {
	parts := strings.Split(s, "-T")
	if len(parts) != 2 {
		return portfolio.Date{}, fmt.Errorf("invalid quarterly date format: %q", s)
	}

	year, err := strconv.Atoi(parts[0])
	if err != nil {
		return portfolio.Date{}, fmt.Errorf("invalid year in quarterly date %q: %w", s, err)
	}

	quarter, err := strconv.Atoi(parts[1])
	if err != nil || quarter < 1 || quarter > 4 {
		return portfolio.Date{}, fmt.Errorf("invalid quarter in quarterly date %q: %w", s, err)
	}

	// The date represents the end of the quarter.
	month := time.Month(quarter * 3)
	return portfolio.NewDate(year, month+1, 0), nil
}

// parseSeries reads the INSEE CSV format from an io.Reader.
func parseSeries(r io.Reader) (*Series, error) {

	reader := csv.NewReader(r)
	reader.Comma = ';'
	reader.FieldsPerRecord = -1 // Allow variable number of fields

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read csv: %w", err)
	}

	// records are in three lines and in columns, values start at col 5 (index 4)
	// something like this:
	/*
	   Libellé     "idBank"    "Mises à jour"      "Période"   "2026-T2"   "2026-T1"   "2025-T4"       "2025-T3"
	   lib         "010567069" "20/11/2025 08:45"  ""          ""          ""          ""              "136.9"
	   Codes       ""          ""                  ""          ""          ""          ""               "P"

	*/

	if len(records) < 5 {
		return nil, fmt.Errorf("INSEE csv serie is empty with  %d colums. %v", len(records), records)
	}

	series := &Series{
		Libelle: records[0][1],
		IDBank:  records[1][1],
		Values:  make(map[portfolio.Date]float64),
	}

	// parse the updated date.
	series.LastUpdate, err = time.Parse("02/01/2006 15:04", records[2][1])
	if err != nil {
		return nil, fmt.Errorf("failed to parse last update date %q: %w", records[2][1], err)
	}

	// parse each value in the serie.
	for i := 4; i < len(records); i++ {
		if len(records[i]) > 1 && records[i][1] != "" {
			log.Printf("INSEE: %v : %v", records[i][0], records[i][1])
			date, err := parseInseeDate(records[i][0])
			if err != nil {
				// Don't wrap, parseInseeDate provides good context
				return nil, err
			}
			val, err := strconv.ParseFloat(records[i][1], 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse value %q for date %q: %w", records[i][1], records[i][0], err)
			}
			series.Values[date] = val
		}
	}
	return series, nil
}
