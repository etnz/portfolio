# Summary Report

## Purpose

The `pcs summary` report provides an overview of your investment strategy's performance, focusing on metrics that reflect the growth of your capital independent of cash contributions or withdrawals. It helps answer the question: "How well is my investment *strategy* performing as a percentage, regardless of when I added or removed cash?"

## Key Metrics & Calculation Methods

*   **Time-Weighted Return (TWR):** This is the primary metric for evaluating investment strategy performance. It measures the compound rate of growth of an investment portfolio over a specified period, eliminating the distorting effects of cash inflows and outflows. It is calculated by geometrically linking the returns of individual sub-periods.
*   **Total Portfolio Value (Start):** The market value of all holdings (securities and cash) at the beginning of the reporting period.
*   **Total Portfolio Value (End):** The market value of all holdings (securities and cash) at the end of the reporting period.
*   **Net Cash Flow:** The sum of all deposits and withdrawals within the reporting period.
*   **Market Gains/Losses:** The change in value of your securities due to price fluctuations during the period.
*   **Realized Gains/Losses:** The profit or loss from selling securities during the period.

# Summary Report

## Purpose

The `pcs summary` report provides an overview of your investment strategy's performance, focusing on metrics that reflect the growth of your capital independent of cash contributions or withdrawals. It helps answer the question: "How well is my investment *strategy* performing as a percentage, regardless of when I added or removed cash?"

## Key Metrics & Calculation Methods

*   **Time-Weighted Return (TWR):** This is the primary metric for evaluating investment strategy performance. It measures the compound rate of growth of an investment portfolio over a specified period, eliminating the distorting effects of cash inflows and outflows. It is calculated by geometrically linking the returns of individual sub-periods.
*   **Total Portfolio Value (Start):** The market value of all holdings (securities and cash) at the beginning of the reporting period.
*   **Total Portfolio Value (End):** The market value of all holdings (securities and cash) at the end of the reporting period.
*   **Net Cash Flow:** The sum of all deposits and withdrawals within the reporting period.
*   **Market Gains/Losses:** The change in value of your securities due to price fluctuations during the period.
*   **Realized Gains/Losses:** The profit or loss from selling securities during the period.

## Scenarios

### Standard Case

This scenario demonstrates the calculation of Time-Weighted Return (TWR) for a simple investment over different periods.

```bash setup
pcs deposit -d 2025-01-01 -c EUR -a 10000
pcs deposit -d 2025-01-01 -c USD -a 2000
pcs declare -d 2025-01-01 -s MSFT -id US0378331005.XNAS -c USD
pcs buy -d 2025-01-02 -s MSFT -q 10 -a 1000
pcs add-security -s MSFT -id US0378331005.XNAS -c USD
pcs add-security -s EURUSD -id EURUSD -c USD
pcs update-security -id US0378331005.XNAS -d 2025-01-02 -p 100
pcs update-security -id EURUSD -d 2025-01-02 -p 1.1
pcs update-security -id US0378331005.XNAS -d 2025-01-03 -p 105
pcs update-security -id EURUSD -d 2025-01-03 -p 1.1
pcs update-security -id US0378331005.XNAS -d 2025-01-08 -p 110
pcs update-security -id EURUSD -d 2025-01-08 -p 1.1
pcs update-security -id US0378331005.XNAS -d 2025-01-31 -p 115
pcs update-security -id EURUSD -d 2025-01-31 -p 1.1
```

```bash run
pcs summary -d 2025-01-31 -c EUR
```

```console check
Portfolio Summary on 2025-01-31
-------------------------------------------
Total Market Value: 11954.55 EUR

Performance:
  Day 31:         +0.38%
  Week 5:         +0.38%
  January:           N/A
  Q1:                N/A
  2025:              N/A
  Inception:         N/A
```

### Surprising Case

This scenario demonstrates how a large cash deposit can lead to a positive change in total portfolio value, even if the Time-Weighted Return (TWR) for the period is negative due to market losses.

```bash setup
pcs deposit -d 2025-01-01 -c EUR -a 10000
pcs deposit -d 2025-01-01 -c USD -a 2000
pcs declare -d 2025-01-01 -s MSFT -id US0378331005.XNAS -c USD
pcs buy -d 2025-01-02 -s MSFT -q 10 -a 1000
pcs add-security -s MSFT -id US0378331005.XNAS -c USD
pcs add-security -s EURUSD -id EURUSD -c USD
pcs update-security -id US0378331005.XNAS -d 2025-01-02 -p 100
pcs update-security -id EURUSD -d 2025-01-02 -p 1.1
pcs update-security -id US0378331005.XNAS -d 2025-01-03 -p 95
pcs update-security -id EURUSD -d 2025-01-03 -p 1.1
pcs deposit -d 2025-01-03 -c EUR -a 5000
```

```bash run
pcs summary -d 2025-01-03 -c EUR
```

```console check
Portfolio Summary on 2025-01-03
-------------------------------------------
Total Market Value: 16772.73 EUR

Performance:
  Day 3:          -0.38%
  Week 1:            N/A
  January:           N/A
  Q1:                N/A
  2025:              N/A
  Inception:         N/A
```

**Explanation:**

On 2025-01-03, the MSFT stock price dropped, leading to a negative daily return (-4.55%). However, a large cash deposit of 5000 EUR on the same day significantly increased the total portfolio value. The Time-Weighted Return (TWR) correctly reflects the negative performance of the investment itself, as it neutralizes the effect of cash flows. The total portfolio value, on the other hand, shows a substantial increase due to the cash injection. This demonstrates that TWR is a more accurate measure of investment strategy performance, as it separates the impact of market movements from the impact of investor behavior (deposits/withdrawals).

