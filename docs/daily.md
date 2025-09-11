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

### Basic Portfolio Growth

This scenario sets up a basic dual-currency investment portfolio by funding it with cash in EUR and USD, and then making an initial purchase of Microsoft (MSFT) shares. This prepares the portfolio to demonstrate how `pcs` tracks daily gains and losses.

```bash setup
# Set up the market data.
pcs add-security -s EURUSD -id EURUSD -c USD
pcs add-security -s MSFT -id US0378331005.XNAS -c USD
# Fund the portfolio with EUR and USD.
pcs deposit -d 2025-01-01 -c EUR -a 10000
pcs deposit -d 2025-01-01 -c USD -a 2000
# Add stock to the ledger and make the first buy transaction.
pcs declare -d 2025-01-01 -s MSFT -id US0378331005.XNAS -c USD
pcs buy     -d 2025-01-02 -s MSFT -q 10 -a 1000
# Manually updating market data to explicitly show price changes.
# In a real-world daily routine, `pcs fetch-security` would automate this.
pcs update-security -id US0378331005.XNAS -d 2025-01-02 -p 100
pcs update-security -id US0378331005.XNAS -d 2025-01-03 -p 105
pcs update-security -id EURUSD -d 2025-01-02 -p 1.1
pcs update-security -id EURUSD -d 2025-01-03 -p 1.1
```

On the next day, the portfolio is showing some gains.


```bash run
pcs daily -d 2025-01-03 -c EUR
```

```console check
# Daily Report
  
  Report for 2025-01-03
  
   **Value**            | **€11,863.63** 
  ----------------------|----------------
   Value at Prev. Close |     €11,818.18 
  
  ## Breakdown of Change
  
   **Total Day's Gain** | **+€45.45** 
  ----------------------|-------------
   Unrealized Market    |     +€45.45 
  
  ## Active Assets
  
   Ticker    | Gain / Loss |     Change 
  -----------|-------------|------------
   MSFT      |      €45.45 |     +5.00% 
   **Total** | **+€45.45** | **+0.38%**
```

### Cash Flow Impact on Daily Gains

This scenario illustrates how a significant cash inflow can lead to an overall positive portfolio value change, even when market performance for securities is negative. It highlights the distinction between market gains/losses and the impact of cash movements on total portfolio value.

```bash setup
# Set up the market data.
pcs add-security -s EURUSD -id EURUSD -c USD
pcs add-security -s MSFT -id US0378331005.XNAS -c USD
# Fund the portfolio with EUR and USD.
pcs deposit -d 2025-01-01 -c EUR -a 10000
pcs deposit -d 2025-01-01 -c USD -a 2000
# Add stock to the ledger and make the first buy transaction.
pcs declare -d 2025-01-01 -s MSFT -id US0378331005.XNAS -c USD
pcs buy     -d 2025-01-02 -s MSFT -q 10 -a 1000
# Manually updating market data to explicitly show price changes.
# In a real-world daily routine, `pcs fetch-security` would automate this.
pcs update-security -id US0378331005.XNAS -d 2025-01-02 -p 100
pcs update-security -id US0378331005.XNAS -d 2025-01-03 -p 95
pcs update-security -id EURUSD -d 2025-01-02 -p 1.1
pcs update-security -id EURUSD -d 2025-01-03 -p 1.1
# An additional deposit is made to demonstrate its effect.
pcs deposit -d 2025-01-03 -c EUR -a 500
```

Observe how the total daily gain remains positive, and the breakdown clarifies the contributing factors.


```bash run
pcs daily -d 2025-01-03 -c EUR
```

```console check
# Daily Report
  
  Report for 2025-01-03
  
   **Value**            | **€12,272.72** 
  ----------------------|----------------
   Value at Prev. Close |     €11,818.18 
  
  ## Breakdown of Change
  
   **Total Day's Gain** | **+€454.54** 
  ----------------------|--------------
   Unrealized Market    |      -€45.45 
   Net Cash Flow        |     +€500.00 
  
  ## Active Assets
  
   Ticker    | Gain / Loss |     Change 
  -----------|-------------|------------
   MSFT      |     -€45.45 |     -5.00% 
   **Total** | **-€45.45** | **+3.85%** 
  
  ## Intraday's Transactions
  
  1. Deposited €500.00
 ```