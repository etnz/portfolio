## Performance Calculation

`pcs` calculates portfolio performance using the **Time-Weighted Return (TWR)** method. TWR measures the compound growth rate of a portfolio, removing the distorting effects of cash flows. This is the industry standard for comparing investment manager performance.

### Methodology

The TWR method breaks the total investment period into smaller sub-periods, using each external cash flow (e.g., a deposit or withdrawal) as a dividing point. A simple rate of return is calculated for each sub-period.

On the day of a cash flow, the portfolio's value is determined *just before* the cash flow occurs. This closing value is used to calculate the return for the sub-period that is ending. The cash flow is then applied, and the resulting new portfolio value becomes the starting point for the next sub-period.

Finally, all the sub-period returns are geometrically linked (compounded) together to determine the overall time-weighted return for the entire period.

### Further Reading

For a more detailed mathematical explanation, you can refer to the Wikipedia article on the subject:

*   [Time-weighted return on Wikipedia](https://en.wikipedia.org/wiki/Time-weighted_return)
