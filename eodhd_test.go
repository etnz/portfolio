package portfolio

import (
	"testing"
)

const EodhdApiDemoKey = "67adc13417e148.00145034"

func Test_eodhdDailyFrom(t *testing.T) {

	_, prices, err := eodhdDaily(EodhdApiDemoKey, "MCD.US", Today().Add(-10), Today().Add(-1))
	if err != nil {
		t.Errorf("eodhdDailyFrom() unexpected error = %v", err)
	}
	if prices.Len() == 0 {
		t.Error("eodhdDailyFrom() no prices returned")
	}
}
