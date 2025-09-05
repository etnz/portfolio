package portfolio

import (
	"fmt"
	"log"

	"github.com/Rhymond/go-money"
	"github.com/shopspring/decimal"
)

type Quantity struct {
	value decimal.Decimal
}

func (q Quantity) Equals(quantity Quantity) bool {
	return q.value.Equal(quantity.value)
}

func NewQuantity(value decimal.Decimal) Quantity {
	return Quantity{value: value}
}

func NewQuantityFromFloat(val float64) Quantity {
	return NewQuantity(decimal.NewFromFloat(val))
}

func (q Quantity) String() string {
	return q.value.String()
}

func (q Quantity) IsZero() bool {
	return q.value.IsZero()
}

// Money represents a monetary value.
type Money struct {
	value *money.Money
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

func NewMoneyFromFloat(amount float64, currency string) Money {
	return NewMoney(decimal.NewFromFloat(amount), currency)
}

// String returns the string representation of the money value.
func (m Money) String() string {
	return m.value.Display()
}

func (m Money) Equals(other Money) bool {
	eq, err := m.value.Equals(other.value)
	return err == nil && eq
}

func (m Money) IsZero() bool {
	return m.value.IsZero()
}
func (m Money) IsPositive() bool {
	return m.value.IsPositive()
}

func (m Money) IsNegative() bool {
	return m.value.IsNegative()
}
func (m Money) Sub(n Money) Money {
	r, err := m.value.Subtract(n.value)
	if err != nil {
		log.Fatalf("invalid money operation: %v", err)
	}
	return Money{r}
}

func (m Money) AsFloat() float64 {
	return m.value.AsMajorUnits()
}

// SignedString returns the string representation of the money value with a sign.
func (m Money) SignedString() string {
	if m.value.IsPositive() {
		return "+" + m.value.Display()
	}
	return m.value.Display()
}

type Percent float64

func (p Percent) String() string {
	return fmt.Sprintf("%.2f%%", p)
}

func (p Percent) SignedString() string {
	return fmt.Sprintf("%+.2f%%", p)
}
