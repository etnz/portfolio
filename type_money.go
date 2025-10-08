package portfolio

import (
	"github.com/Rhymond/go-money"
	"github.com/shopspring/decimal"
)

// Money represents a monetary value.
type Money struct {
	value      decimal.Decimal // as major unit value
	cur        string
	fractional bool // true to persist in full digits
}

func M[T float32 | float64 | int | int32 | int64 | uint | uint32 | uint64 | decimal.Decimal](value T, currency string) Money {
	return Money{value: newDecimal(value), cur: currency}
}

// functions that requires the full currency

// currency returns the money's currency
func (m Money) currency() money.Currency {
	// to get a never nil currency I need to call the Money constructor
	return *money.New(0, m.cur).Currency()
}

// String returns the string representation of the money value.
func (m Money) String() string {
	cur := m.currency()
	dec := m.value.Shift(int32(cur.Fraction))
	return cur.Formatter().Format(dec.IntPart())
}

// Simple wrapper around money.Money

func (m Money) Currency() string                { return m.cur }
func (m Money) Equal(n Money) bool              { return m.value.Equal(n.value) && m.cur == n.cur }
func (m Money) IsZero() bool                    { return m.value.IsZero() }
func (m Money) IsPositive() bool                { return m.value.IsPositive() }
func (m Money) IsNegative() bool                { return m.value.IsNegative() }
func (m Money) LessThan(amount Money) bool      { return m.value.LessThan(amount.value) }
func (m Money) LessThanOrEqual(n Money) bool    { return m.value.LessThanOrEqual(n.value) }
func (m Money) GreaterThan(n Money) bool        { return m.value.GreaterThan(n.value) }
func (m Money) GreaterThanOrEqual(n Money) bool { return m.value.GreaterThanOrEqual(n.value) }
func (m Money) Neg() Money                      { return Money{value: m.value.Neg(), cur: m.cur} }
func (m Money) Mul(n Quantity) Money            { return Money{value: m.value.Mul(n.value), cur: m.cur} }
func (m Money) Div(n Quantity) Money            { return Money{value: m.value.Div(n.value), cur: m.cur} }
func (m Money) DivPrice(n Money) Quantity       { return Quantity{value: m.value.Div(n.value)} }

// binary operators.
func (m Money) Add(n Money) Money { return Money{value: m.value.Add(n.value), cur: cur(m, n)} }
func (m Money) Sub(n Money) Money { return Money{value: m.value.Sub(n.value), cur: cur(m, n)} }

// makes the "" currency totally weak.
func cur(A, B Money) string {
	if A.cur == "" {
		return B.cur
	}
	if B.cur == "" {
		return A.cur
	}
	if A.cur != B.cur {
		panic("currency mismatch" + A.cur + "!=" + B.cur)
	}
	return A.cur
}

// Deprecated: AsFloat should no longer be used, the purpose is to keep the calculation exact.
func (m Money) AsFloat() float64 { return m.value.InexactFloat64() }

// SignedString returns the string representation of the money value with a sign.
// 0 is represented as a ""
func (m Money) SignedString() string {
	if m.value.IsZero() {
		return "-"
	}
	if m.value.IsPositive() {
		return "+" + m.String()
	}
	return m.String()
}

// exact return a copy of money that will be persisted with all the digits.
func (m Money) exact() Money {
	m.fractional = true
	return m
}

func (m Money) MarshalJSON() ([]byte, error) {
	var w jsonObjectWriter
	w.Optional("currency", m.cur)
	// it was rounded to 2, which is ok in USD and EUR but should at the very least use cur.Fraction.
	// however, money is used for dividend per share that can be fractional.
	rounded := m.value // no rounding by default
	if !m.fractional {
		rounded = m.value.Round(int32(m.currency().Fraction))
	}
	w.Append("amount", rounded)
	return w.MarshalJSON()
}
