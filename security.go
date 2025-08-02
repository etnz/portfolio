package portfolio

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/etnz/portfolio/date"
)

// isinRegex checks for the basic structure: 2 letters, 9 alphanumeric, 1 digit.
var isinRegex = regexp.MustCompile(`^[A-Z]{2}[A-Z0-9]{9}[0-9]$`)

// micRegex checks for the format: 4 uppercase alphanumeric characters.
var micRegex = regexp.MustCompile(`^[A-Z0-9]{4}$`)

// currencyCodeRegex checks for the format: 3 uppercase letters.
var currencyCodeRegex = regexp.MustCompile(`^[A-Z]{3}$`)

// currencyPairRegex checks for the format: 6 uppercase letters (3 for base, 3 for quote).
var currencyPairRegex = regexp.MustCompile(`^[A-Z]{6}$`)

// idCharRegex checks for alphanumeric characters and space, used in Private IDs.
var idCharRegex = regexp.MustCompile(`^[a-zA-Z0-9 ]+$`)

// ID represent the unique identifier of a security. It must follow a specific format.
//
// It can have multiple different types.
//
// MSSI (Market-Specific Security Identifier).
//
// MSSI represents a Market-Specific Security Identifier, a proposed standard for
// creating an unambiguous, composite identifier for a security listed on a
// specific trading venue.
//
// The format is defined as the concatenation of an ISIN and a MIC, separated
// by a FULL STOP character ('.').
//
// Formal Definition: ID = ISIN "." MIC
//
// This type provides a safe way to create, parse, and handle MSSIs, ensuring
// the format is always valid. It is based on two existing ISO standards:
//   - ISO 6166 (ISIN): For identifying the security.
//   - ISO 10383 (MIC): For identifying the market.
//
// CurrencyPair.
//
// CurrencyPair represents a currency pair identifier according to the common market
// convention used in foreign exchange (FX) markets.
//
// The format is a six-character string created by concatenating two three-character
// ISO 4217 currency codes.
//
// Formal Convention: CurrencyPair = <BaseCurrency><QuoteCurrency>
//
// - BaseCurrency: The first 3-letter code, representing the currency being priced.
// - QuoteCurrency: The second 3-letter code, representing the currency used for the price.
//
// This "Base/Quote" terminology is the industry standard because it is
// unambiguous, unlike "From/To" which can change depending on the direction
// of a transaction.
//
// Example: The pair "EURUSD" represents the price of one Euro (EUR) in terms of
// US Dollars (USD).
//
// Private.
//
// Private represents a generic, non-standard identifier.
//
// The format rules for a private ID are designed to prevent ambiguity with other, more
// specific financial identifiers. The rationale is to ensure that a private ID cannot be
// misinterpreted as a Market-Specific Security Identifier (MSSI) or a CurrencyPair.
//
// Rules:
//  1. Must be at least 7 characters long.
//  2. Must only contain alphanumeric characters and space ([ a-zA-Z0-9]).
//  3. Must NOT contain a '.' to avoid confusion with the "ISIN.MIC" format of an MSSI.
//  4. Must NOT be a 6-character, all-uppercase string (implicitly covered by the length rule).
type ID string

// NewMSSI creates a new MSSI from its constituent parts after basic validation.
func NewMSSI(isin, mic string) (ID, error) {
	if err := ValidateISIN(isin); err != nil {
		return "", fmt.Errorf("invalid ISIN: %w", err)
	}
	if mic == "" {
		return "", errors.New("mic cannot be empty")
	}
	// In a real-world library, you would add regex validation for ISIN and MIC formats.
	return ID(fmt.Sprintf("%s.%s", isin, mic)), nil
}

// NewCurrencyPair creates a new CurrencyPair from a base and quote currency code after validation.
func NewCurrencyPair(base, quote string) (ID, error) {
	if !currencyCodeRegex.MatchString(base) {
		return "", fmt.Errorf("invalid base currency format: must be 3 uppercase letters, got %q", base)
	}
	if !currencyCodeRegex.MatchString(quote) {
		return "", fmt.Errorf("invalid quote currency format: must be 3 uppercase letters, got %q", quote)
	}
	return ID(base + quote), nil
}

// NewPrivate validates that a string is a valid ID.
func NewPrivate(s string) (ID, error) {
	// Rule 1: Must be at least 7 characters long.
	// This also implicitly invalidates 6-character Currency Pairs.
	if len(s) < 7 {
		return "", fmt.Errorf("invalid id: must be at least 7 characters long, got %d", len(s))
	}

	// Rule 3: Must NOT contain a '.'
	if strings.Contains(s, ".") {
		return "", fmt.Errorf("invalid id: must not contain a '.' (resembles an MSSI)")
	}

	// Rule 2: Must be alphanumeric or a space.
	if !idCharRegex.MatchString(s) {
		return "", fmt.Errorf("invalid id: must only contain alphanumeric characters and spaces")
	}

	return ID(s), nil
}

// ValidateISIN checks if a string is a validly formatted ISIN.
// It returns nil if valid, or a descriptive error if invalid.
func ValidateISIN(isin string) error {
	// 1. Length validation
	if len(isin) != 12 {
		return fmt.Errorf("invalid length: must be 12 characters, got %d", len(isin))
	}

	// 2. Format validation
	if !isinRegex.MatchString(isin) {
		return fmt.Errorf("invalid format: must be 2 uppercase letters, 9 alphanumeric chars, and 1 digit")
	}

	// 3. Convert letters to numbers for check digit calculation
	var numericStr strings.Builder
	for _, char := range isin[:11] {
		if char >= 'A' && char <= 'Z' {
			numericStr.WriteString(strconv.Itoa(int(char - 'A' + 10)))
		} else {
			numericStr.WriteRune(char)
		}
	}

	// 4. Apply a variation of the Luhn algorithm
	sum := 0
	isSecond := true
	digits := numericStr.String()
	for i := len(digits) - 1; i >= 0; i-- {
		digit, _ := strconv.Atoi(string(digits[i]))

		if isSecond {
			digit *= 2
		}

		sum += (digit / 10) + (digit % 10)
		isSecond = !isSecond
	}

	// 5. Validate the check digit
	expectedCheckDigit := (10 - (sum % 10)) % 10
	actualCheckDigit, _ := strconv.Atoi(string(isin[11]))

	if expectedCheckDigit != actualCheckDigit {
		return fmt.Errorf("invalid check digit: expected %d, got %d", expectedCheckDigit, actualCheckDigit)
	}

	// If all checks pass, the ISIN is valid
	return nil
}

// ValidateMIC checks if a string conforms to the MIC (ISO 10383) format.
// It returns nil if valid, or a descriptive error if invalid.
// Note: This validates the format only, not whether the MIC is officially registered.
func ValidateMIC(mic string) error {
	// 1. Length validation
	if len(mic) != 4 {
		return fmt.Errorf("invalid length: must be 4 characters, got %d", len(mic))
	}

	// 2. Format validation
	if !micRegex.MatchString(mic) {
		return fmt.Errorf("invalid format: must be 4 uppercase alphanumeric characters")
	}

	// If all checks pass, the MIC format is valid
	return nil
}

// MSSI validates the overall "ISIN.MIC" format and then validates each
// component using the ValidateISIN and ValidateMIC functions.
func (id ID) MSSI() (isin string, mic string, err error) {
	// 1. Basic structural validation
	parts := strings.Split(string(id), ".")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid format: MSSI must contain exactly one '.', got %q", id)
	}
	isin = parts[0]
	mic = parts[1]

	// 2. Validate the extracted ISIN component
	if err := ValidateISIN(isin); err != nil {
		return "", "", fmt.Errorf("invalid ISIN part: %w", err)
	}

	// 3. Validate the extracted MIC component
	if err := ValidateMIC(mic); err != nil {
		return "", "", fmt.Errorf("invalid MIC part: %w", err)
	}

	return isin, mic, nil
}

// ISIN returns the ISIN part of the identifier or an empty string if the ID is not an MSSI.
func (id ID) ISIN() string {
	isin, _, _ := id.MSSI()
	return isin
}

// MIC returns the Market Identifier Code part of the identifier or an empty string if the ID is not an MSSI.
func (id ID) MIC() string {
	_, mic, _ := id.MSSI()
	return mic
}

// CurrencyPair validates a 6-character string and extracts the base and quote
// components. It returns an error if the format is invalid.
func (id ID) CurrencyPair() (base string, quote string, err error) {
	if len(id) != 6 {
		return "", "", fmt.Errorf("invalid length: currency pair must be 6 characters, got %d", len(id))
	}
	if !currencyPairRegex.MatchString(string(id)) {
		return "", "", fmt.Errorf("invalid format: currency pair must be 6 uppercase letters")
	}

	// On success, extract and return the components.
	base = string(id)[:3]
	quote = string(id)[3:]
	return base, quote, nil
}

// Base returns the base currency of the currency pair or an empty string if the ID is not a valid currency pair.
func (id ID) Base() string {
	base, _, _ := id.CurrencyPair()
	return base
}

// Quote returns the quote currency of the currency pair or an empty string if the ID is not a valid currency pair.
func (id ID) Quote() string {
	_, quote, _ := id.CurrencyPair()
	return quote
}

// Private validates that a string is a valid Private ID.
func (id ID) Private() (private string, err error) {
	_, err = NewPrivate(string(id))
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

// String implements the fmt.Stringer interface.
func (id ID) String() string {
	return string(id)
}

// Security represent a publicly or privately tradeable asset, stock, ETF, house.
type Security struct {
	id     ID
	ticker string                // The ticker used in portfolio and human friendly persistence format.
	prices date.History[float64] // the price historical value.
}

func (s *Security) ID() ID {
	return s.id
}

func (s *Security) Ticker() string {
	return s.ticker
}

func (s *Security) Prices() *date.History[float64] {
	return &s.prices
}
