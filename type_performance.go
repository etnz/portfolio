package portfolio

// Performance holds the starting value and the calculated change for a specific range.
type Performance struct {
	Start, End Money
	Return     Percent // TWR return if available or price change
}

func NewPerformance(start, end Money) Performance {
	return Performance{
		Start: start,
		End:   end,
	}
}
func NewPerformanceWithReturn(start, end Money, ret Percent) Performance {
	return Performance{
		Start: start,
		End:   end,
		Return: ret,
	}
}

func (p Performance) Change() Money {
	return p.End.Sub(p.Start)
}

func (p Performance) Percent() Percent {
	return Percent(100 * p.Change().AsFloat() / p.Start.AsFloat())
}
