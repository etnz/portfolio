package portfolio

import "fmt"

type Percent float64

func (p Percent) Equal(q Percent) bool {
	// it has to be compared with some precision
	const precision = 0.0001
	diff := p - q
	if diff < 0 {
		diff = -diff
	}
	return diff < precision
}

func (p Percent) String() string {
	return fmt.Sprintf("%.2f%%", p)
}

func (p Percent) SignedString() string {
	res := fmt.Sprintf("%+.2f%%", p)
	if res == "+0.00%" {
		return "-"
	}
	return res
}
