package portfolio

import (
	"testing"

	"github.com/etnz/portfolio/date"
)

func Test_eodhdDailyFrom(t *testing.T) {

	_, prices, err := eodhdDaily(EodhdApiDemoKey, "MCD.US", date.Today().Add(-10), date.Today().Add(-1))
	if err != nil {
		t.Errorf("eodhdDailyFrom() unexpected error = %v", err)
	}
	if prices.Len() == 0 {
		t.Error("eodhdDailyFrom() no prices returned")
	}
}
