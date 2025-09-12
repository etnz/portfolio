package eodhd

import (
	"testing"

	"github.com/etnz/portfolio"
)

const EodhdApiDemoKey = "67adc13417e148.00145034"

func Test_eodhdDailyFrom(t *testing.T) {

	prices := make(map[portfolio.Date]float64)
	err := eodhdDaily(EodhdApiDemoKey, "MCD.US", portfolio.Today().Add(-10), portfolio.Today().Add(-1), nil, prices)
	if err != nil {
		t.Errorf("eodhdDailyFrom() unexpected error = %v", err)
	}
	if len(prices) == 0 {
		t.Error("eodhdDailyFrom() no prices returned")
	}
}
