package portfolio

import (
	"fmt"

	"github.com/Rhymond/go-money"
	"github.com/shopspring/decimal"
)

// Money represents a monetary value.
type Money struct {
	*money.Money
}

// NewMoney creates a new Money instance from a decimal.Decimal.
func NewMoney(amount decimal.Decimal, currency string) Money {
	// Find the currency first.
	cur := money.GetCurrency(currency)
	if cur == nil {
		return Money{}
	}

	factor, _ := decimal.NewFromInt(10).PowInt32(int32(cur.Fraction))
	amount = amount.Mul(factor)
	return Money{money.New(amount.IntPart(), currency)}
}

// String returns the string representation of the money value.
func (m Money) String() string {
	return m.Money.Display()
}

// SignedString returns the string representation of the money value with a sign.
func (m Money) SignedString() string {
	if m.IsPositive() {
		return "+" + m.Money.Display()
	}
	return m.Money.Display()
}

type Percent float64

func (p Percent) String() string {
	return fmt.Sprintf("%.2f%%", p)
}

func (p Percent) SignedString() string {
	return fmt.Sprintf("%+.2f%%", p)
}
