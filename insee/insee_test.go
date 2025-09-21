package insee

import (
	"strings"
	"testing"
	"time"

	"github.com/etnz/portfolio"
)

func TestParseSeries(t *testing.T) {
	csvData := `"Libellé";"Indice des prix des logements anciens - Province : agglomérations de moins de 10 000 habitants et zones rurales - Appartements - Base 100 en moyenne annuelle 2015 - Série CVS";"Codes"
"idBank";"010567069";""
"Dernière mise à jour";"28/08/2025 08:45";""
"Période";"";""
"2025-T4";"";""
"2025-T3";"";""
"2025-T2";"135.2";"P"
"2025-T1";"135.6";"A"
"2024-T4";"133.4";"A"
`

	reader := strings.NewReader(csvData)
	series, err := parseSeries(reader)
	if err != nil {
		t.Fatalf("parseSeries() failed: %v", err)
	}

	expectedLibelle := "Indice des prix des logements anciens - Province : agglomérations de moins de 10 000 habitants et zones rurales - Appartements - Base 100 en moyenne annuelle 2015 - Série CVS"
	if series.Libelle != expectedLibelle {
		t.Errorf("got Libelle %q, want %q", series.Libelle, expectedLibelle)
	}

	expectedIDBank := "010567069"
	if series.IDBank != expectedIDBank {
		t.Errorf("got IDBank %q, want %q", series.IDBank, expectedIDBank)
	}

	expectedLastUpdate := time.Date(2025, 8, 28, 8, 45, 0, 0, time.UTC)
	if !series.LastUpdate.Equal(expectedLastUpdate) {
		t.Errorf("got LastUpdate %v, want %v", series.LastUpdate, expectedLastUpdate)
	}

	if len(series.Values) != 3 {
		t.Errorf("got %d values, want 3", len(series.Values))
	}

	dateT2_2025 := portfolio.NewDate(2025, 6, 30)
	if val, ok := series.Values[dateT2_2025]; !ok || val != 135.2 {
		t.Errorf("for date %v, got %f, want 135.2", dateT2_2025, val)
	}

	dateT4_2024 := portfolio.NewDate(2024, 12, 31)
	if val, ok := series.Values[dateT4_2024]; !ok || val != 133.4 {
		t.Errorf("for date %v, got %f, want 133.4", dateT4_2024, val)
	}
}

func TestParseSeries_Errors(t *testing.T) {
	testCases := []struct {
		name    string
		csvData string
		wantErr string
	}{
		{
			name: "bad last update date",
			csvData: `"Libellé";"..."
"idBank";"..."
"Dernière mise à jour";"not-a-date"
"Période";""
`,
			wantErr: "failed to parse last update date",
		},
		{
			name: "bad quarterly date",
			csvData: `"Libellé";"..."
"idBank";"..."
"Dernière mise à jour";"28/08/2025 08:45"
"Période";""
"2025-T5";"135.2"`,
			wantErr: "invalid quarter in quarterly date",
		},
		{
			name: "bad value",
			csvData: `"Libellé";"..."
"idBank";"..."
"Dernière mise à jour";"28/08/2025 08:45"
"Période";""
"2025-T2";"not-a-float"`,
			wantErr: "failed to parse value",
		},
		{
			name: "not enough records",
			csvData: `"Libellé";"..."
"idBank";"..."`,
			wantErr: "not enough records in csv",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := strings.NewReader(tc.csvData)
			_, err := parseSeries(reader)
			if err == nil {
				t.Fatalf("parseSeries() expected an error, but got none")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("parseSeries() error = %q, want to contain %q", err, tc.wantErr)
			}
		})
	}
}

func TestGetSeries(t *testing.T) {
	// This is an integration test that hits the live INSEE server.
	if testing.Short() {
		t.Skip("skipping integration test in short mode.")
	}

	idBank := "001763825" // Indice des prix à la consommation
	from := portfolio.NewDate(2025, 1, 1)
	to := portfolio.NewDate(2023, 12, 31)

	series, err := getSeries(idBank, from, to)
	if err != nil {
		t.Fatalf("getSeries() failed: %v", err)
	}

	if series.IDBank != idBank {
		t.Errorf("got IDBank %q, want %q", series.IDBank, idBank)
	}

	if len(series.Values) == 0 {
		t.Error("expected to get some values, but got none")
	}
}
