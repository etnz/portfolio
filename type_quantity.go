package portfolio

import "github.com/shopspring/decimal"

// newDecimal is a convenient factory for decimal.Decimal
func newDecimal[T float32 | float64 | int | int32 | int64 | uint | uint32 | uint64 | decimal.Decimal](value T) decimal.Decimal {
	switch v := any(value).(type) {
	case decimal.Decimal:
		return v
	case float32:
		return decimal.NewFromFloat32(v)
	case float64:
		return decimal.NewFromFloat(v)
	case int:
		return decimal.NewFromInt32(int32(v))
	case int32:
		return decimal.NewFromInt32(v)
	case int64:
		return decimal.NewFromInt(v)
	case uint:
		return decimal.NewFromUint64(uint64(v))
	case uint32:
		return decimal.NewFromUint64(uint64(v))
	case uint64:
		return decimal.NewFromUint64(v)
	default:
		panic("unsupported type")
	}

}

type Quantity struct {
	value decimal.Decimal
}

func Q[T float32 | float64 | int | int32 | int64 | uint | uint32 | uint64 | decimal.Decimal](value T) Quantity {
	return Quantity{value: newDecimal(value)}
}

// Deprecated: MulPrice use Price.Mul instead
// func (t Quantity) MulPrice(m Money) Money { return m.Mul(t) }

func (t Quantity) Equal(p Quantity) bool           { return t.value.Equal(p.value) }
func (t Quantity) LessThan(quantity Quantity) bool { return t.value.LessThan(quantity.value) }
func (t Quantity) Div(p Quantity) Quantity         { return Quantity{value: t.value.Div(p.value)} }
func (t Quantity) Mul(p Quantity) Quantity         { return Quantity{value: t.value.Mul(p.value)} }
func (t Quantity) Add(p Quantity) Quantity         { return Quantity{value: t.value.Add(p.value)} }
func (t Quantity) Sub(p Quantity) Quantity         { return Quantity{value: t.value.Sub(p.value)} }
func (t Quantity) GreaterThan(p Quantity) bool     { return t.value.GreaterThan(p.value) }
func (t Quantity) IsNegative() bool                { return t.value.IsNegative() }
func (t Quantity) IsPositive() bool                { return t.value.IsPositive() }
func (t Quantity) IsZero() bool                    { return t.value.IsZero() }
func (q Quantity) String() string                  { return q.value.String() }

// MarshalJSON implements the json.Marshaler interface for baseCmd.
func (t Quantity) MarshalJSON() ([]byte, error) {
	return t.value.MarshalJSON()
}
func (t *Quantity) UnmarshalJSON(decimalBytes []byte) error {
	return t.value.UnmarshalJSON(decimalBytes)
}
