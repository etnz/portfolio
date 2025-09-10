package portfolio

import "fmt"

type Percent float64

func (p Percent) String() string {
	return fmt.Sprintf("%.2f%%", p)
}

func (p Percent) SignedString() string {
	return fmt.Sprintf("%+.2f%%", p)
}
