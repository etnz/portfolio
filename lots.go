package portfolio

import (
	"github.com/shopspring/decimal"
)

// lot represents a single purchase of a security.
// lot represents a single purchase of a security, used for cost basis calculations.
type lot struct {
	Date     Date
	Quantity Quantity
	Cost     Money // Total cost of the lot (quantity * price)
}

type lots []lot

// fifoCostOfSelling calculates the cost of selling a quantity of shares using FIFO.
func (l lots) fifoCostOfSelling(quantityToSell Quantity) Money {
	var costOfSoldShares Money

	for _, currentLot := range l {
		if currentLot.Quantity.GreaterThan(quantityToSell) {
			// Partial sale from this lot
			costOfSoldPortion := currentLot.Cost.Mul(quantityToSell).Div(currentLot.Quantity)
			costOfSoldShares = costOfSoldShares.Add(costOfSoldPortion)
			return costOfSoldShares
		} else {
			// Full sale of this lot
			costOfSoldShares = costOfSoldShares.Add(currentLot.Cost)
			quantityToSell = quantityToSell.Sub(currentLot.Quantity)
		}
	}
	return costOfSoldShares
}

// sell reduces the available lots by a given quantity to sell using the FIFO method.
func (l lots) sell(quantityToSell Quantity) lots {
	var remainingLots lots

	for _, currentLot := range l {
		if quantityToSell.IsZero() {
			remainingLots = append(remainingLots, currentLot)
			continue
		}

		if currentLot.Quantity.GreaterThan(quantityToSell) {
			// Partial sale from this lot
			costOfSoldPortion := currentLot.Cost.Mul(quantityToSell).Div(currentLot.Quantity)
			newLot := lot{
				Date:     currentLot.Date,
				Quantity: currentLot.Quantity.Sub(quantityToSell),
				Cost:     currentLot.Cost.Sub(costOfSoldPortion),
			}
			remainingLots = append(remainingLots, newLot)
			quantityToSell = Q(decimal.Zero)
		} else {
			// Full sale of this lot
			quantityToSell = quantityToSell.Sub(currentLot.Quantity)
		}
	}
	return remainingLots
}
