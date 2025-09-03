package portfolio

import (
	"github.com/etnz/portfolio/date"
	"github.com/shopspring/decimal"
)

// lot represents a single purchase of a security.
// lot represents a single purchase of a security, used for cost basis calculations.
type lot struct {
	Date     date.Date
	Quantity decimal.Decimal
	Cost     decimal.Decimal // Total cost of the lot (quantity * price)
}

type lots []lot

// averageCostOfSelling determines the cost of shares sold using the average cost method.
func (l lots) averageCostOfSelling(quantityToSell decimal.Decimal) decimal.Decimal {
	var totalQuantity decimal.Decimal
	var totalCost decimal.Decimal
	for _, currentLot := range l {
		totalQuantity = totalQuantity.Add(currentLot.Quantity)
		totalCost = totalCost.Add(currentLot.Cost)
	}

	if totalQuantity.IsZero() {
		return decimal.Zero // Cannot sell from zero shares, so cost is zero.
	}

	costOfSoldShares := totalCost.Mul(quantityToSell).Div(totalQuantity)
	return costOfSoldShares
}

// fifoCostOfSelling calculates the cost of selling a quantity of shares using FIFO.
func (l lots) fifoCostOfSelling(quantityToSell decimal.Decimal) decimal.Decimal {
	var costOfSoldShares decimal.Decimal

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
func (l lots) sell(quantityToSell decimal.Decimal) lots {
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
			quantityToSell = decimal.Zero
		} else {
			// Full sale of this lot
			quantityToSell = quantityToSell.Sub(currentLot.Quantity)
		}
	}
	return remainingLots
}
