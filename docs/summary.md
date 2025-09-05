# Summary Report

## Purpose

The `pcs summary` report provides an overview of your investment strategy's performance, focusing on metrics that reflect the growth of your capital independent of cash contributions or withdrawals. It helps answer the question: "How well is my investment *strategy* performing as a percentage, regardless of when I added or removed cash?"

## Key Metrics & Calculation Methods

*   **Time-Weighted Return (TWR):** This is the primary metric for evaluating investment strategy performance. It measures the compound rate of growth of an investment portfolio over a specified period, eliminating the distorting effects of cash inflows and outflows. It is calculated by geometrically linking the returns of individual sub-periods. See `pcs topic performance-calculation` for more details.
*   **Total Portfolio Value (Start):** The market value of all holdings (securities and cash) at the beginning of the reporting period.
*   **Total Portfolio Value (End):** The market value of all holdings (securities and cash) at the end of the reporting period.
*   **Net Cash Flow:** The sum of all deposits and withdrawals within the reporting period.
*   **Market Gains/Losses:** The change in value of your securities due to price fluctuations during the period.
*   **Realized Gains/Losses:** The profit or loss from selling securities during the period.

## Scenarios

### Standard Case

This scenario demonstrates the calculation of Time-Weighted Return (TWR) for a simple investment over different periods.

```bash setup
# Set up the market data.
pcs add-security -s EURUSD -id EURUSD -c USD
pcs add-security -s MSFT -id US0378331005.XNAS -c USD
# Manually updating market data to explicitly show price changes.
# In a real-world daily routine, `pcs fetch-security` would automate this.
pcs update-security -d 2025-01-02 -id US0378331005.XNAS -p 100
pcs update-security -d 2025-01-03 -id US0378331005.XNAS -p 105
pcs update-security -d 2025-01-08 -id US0378331005.XNAS -p 110
pcs update-security -d 2025-01-31 -id US0378331005.XNAS -p 115
pcs update-security -d 2025-01-02 -id EURUSD -p 1.1
pcs update-security -d 2025-01-03 -id EURUSD -p 1.1
pcs update-security -d 2025-01-08 -id EURUSD -p 1.1
pcs update-security -d 2025-01-31 -id EURUSD -p 1.1
# Fund the portfolio with EUR and USD.
pcs deposit -d 2025-01-01 -c EUR -a 10000
pcs deposit -d 2025-01-01 -c USD -a 2000
# Add stock to the ledger and make the first buy transaction.
pcs declare -d 2025-01-01 -s MSFT -id US0378331005.XNAS -c USD
pcs buy -d 2025-01-02 -s MSFT -q 10 -a 1000
```

```bash run
pcs summary -d 2025-01-31 -c EUR
```

```console check
# Portfolio Summary on 2025-01-31
  
  Total Market Value: 11954.55 EUR
  
  ## Performance
  
   Period    | Return  
  -----------|---------
   Day 31    | +0.38%  
   Week 5    | +0.38%  
   January   | +10.69% 
   Q1        | +10.69% 
   2025      | +10.69% 
   Inception | +10.69%
```
