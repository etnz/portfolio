# Daily Report

## Purpose

The `pcs daily` report provides a snapshot of your portfolio's performance on a day-by-day basis. It helps you track the daily changes in your portfolio's value, including realized and unrealized gains/losses, and cash movements.

## Key Metrics & Calculation Methods

*   **Date:** The specific date for which the report is generated.
*   **Realized Gain/Loss:** The profit or loss from selling a security on that specific day. It's calculated by subtracting the cost basis of the sold asset from the proceeds of the sale.
*   **Unrealized Gain/Loss:** The change in value of securities held on that specific day. It's the difference between the security's market value at the end of the day and its market value at the beginning of the day (or its cost basis if acquired on that day).
*   **Cash Flow:** The net movement of cash into or out of the portfolio on that specific day, including deposits, withdrawals, and cash from sales or purchases.
*   **Total Gain/Loss:** The sum of the realized and unrealized gains/losses for the day.
*   **Portfolio Value:** The total market value of all holdings (securities and cash) at the end of the day.

## Scenarios

### Standard Case

This scenario demonstrates a typical day with market gains and a simple transaction.

```bash setup
pcs deposit -d 2025-01-01 -c EUR -a 10000
pcs deposit -d 2025-01-01 -c USD -a 2000
pcs declare -d 2025-01-01 -s MSFT -id US0378331005.XNAS -c USD
pcs buy     -d 2025-01-02 -s MSFT -q 10 -a 1000
pcs add-security -s MSFT -id US0378331005.XNAS -c USD
pcs add-security -s EURUSD -id EURUSD -c USD
pcs update-security -id US0378331005.XNAS -d 2025-01-02 -p 100
pcs update-security -id EURUSD -d 2025-01-02 -p 1.1
pcs update-security -id US0378331005.XNAS -d 2025-01-03 -p 105
pcs update-security -id EURUSD -d 2025-01-03 -p 1.1
```

```bash run
pcs daily -d 2025-01-03 -c EUR
```

```console check
# Daily Report
  
   **Value at Day's Close** | **11863.64** 
  --------------------------|--------------
   Value at Prev. Close     |     11818.18 
  
  ## Breakdown of Change
  
   Total Day's Gain  | +45.45 (+0.38%) 
  -------------------|-----------------
   Unrealized Market |          +45.45 
  
  ## Active Assets
  
   Ticker | Gain / Loss | Change 
  --------|-------------|--------
   MSFT   |       45.45 | +5.00% 
```

### Surprising Case

This scenario demonstrates a situation where a large cash deposit on a day with negative market movement can lead to a positive overall portfolio value change, even if the "Total Day's Gain / Loss" is negative due to market losses.

```bash setup
pcs deposit -d 2025-01-01 -c EUR -a 10000
pcs deposit -d 2025-01-01 -c USD -a 2000
pcs declare -d 2025-01-01 -s MSFT -id US0378331005.XNAS -c USD
pcs buy     -d 2025-01-02 -s MSFT -q 10 -a 1000
pcs add-security -s MSFT -id US0378331005.XNAS -c USD
pcs add-security -s EURUSD -id EURUSD -c USD
pcs update-security -id US0378331005.XNAS -d 2025-01-02 -p 100
pcs update-security -id EURUSD -d 2025-01-02 -p 1.1
pcs update-security -id US0378331005.XNAS -d 2025-01-03 -p 95
pcs update-security -id EURUSD -d 2025-01-03 -p 1.1
pcs deposit -d 2025-01-03 -c EUR -a 500
```

```bash run
pcs daily -d 2025-01-03 -c EUR
```

```console check
 # Daily Report
  
   **Value at Day's Close** | **12272.73** 
  --------------------------|--------------
   Value at Prev. Close     |     11818.18 
  
  ## Breakdown of Change
  
   Total Day's Gain  | +454.55 (+3.85%) 
  -------------------|------------------
   Unrealized Market |           -45.45 
   Net Cash Flow     |          +500.00 
  
  ## Active Assets
  
   Ticker | Gain / Loss | Change 
  --------|-------------|--------
   MSFT   |      -45.45 | -5.00% 
  
  ## Today's Transactions
  
  1. deposit
```

**Explanation:**

Despite a negative market gain for MSFT (-45.45 EUR), the overall portfolio value increased significantly due to a large cash deposit (454.55 EUR). The "Total Day's Gain / Loss" reflects the combined effect of market movements and cash flows, showing a positive change in the total portfolio value. This highlights that a positive cash flow can offset negative market performance, leading to an overall increase in portfolio value for the day.

