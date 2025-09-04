# Holding Report

**Command:** `pcs holding`

## Purpose

The holding report provides a comprehensive snapshot of your portfolio's composition and value at a specific point in time. It is designed to answer the fundamental question: "What do I own, and what is it worth?"

This report is essential for understanding your asset allocation, monitoring your investments, and preparing for financial reviews or tax reporting.

## Key Metrics and Their Calculation

The holding report calculates the following key metrics:

*   **Market Value**: The market value of each security is calculated by multiplying the quantity of shares held by the market price. By default, this is the closing price on the report date. However, if the report is for the current day and the `-u` option is activated, the latest intraday price is used. The result is then converted to the reporting currency.
*   **Total Portfolio Value**: This is the sum of the market values of all securities, cash balances, and counterparty accounts, also in the reporting currency.

## Scenarios

### Basic Usage

This scenario demonstrates how to generate a holding report for a specific date.

```bash setup
pcs deposit -d 2025-01-01 -c EUR -a 10000
pcs deposit -d 2025-01-01 -c USD -a 5000
pcs declare -d 2025-01-01 -s MSFT -id US0378331005.XNAS -c USD
pcs buy -d 2025-01-02 -s MSFT -q 10 -a 4000
pcs add-security -s MSFT -id US0378331005.XNAS -c USD
pcs add-security -s EURUSD -id EURUSD -c USD
pcs update-security -id US0378331005.XNAS -d 2025-03-05 -p 420
pcs update-security -id EURUSD -d 2025-03-05 -p 1.1
```

```bash run
pcs holding -d 2025-03-05 -c EUR
```

```console check
# Holding Report on 2025-03-05
  
  Total Portfolio Value: **€14,727.27**
  
  ## Securities
  
   Ticker | Quantity |   Price | Market Value 
  --------|----------|---------|--------------
   MSFT   |       10 | $420.00 |    €3,818.18 
  
  ## Cash
  
   Currency |    Balance |      Value 
  ----------|------------|------------
   EUR      | €10,000.00 | €10,000.00 
   USD      |  $1,000.00 |    €909.09        
 ```

### Counterparty Accounts

This scenario demonstrates how to use counterparty accounts to track liabilities, such as taxes.

```bash run
pcs sell -d 2025-03-06 -s MSFT -q 5 -a 2200
pcs accrue -d 2025-03-06 -c USD -payable TaxAccount -a 60 
```

```bash run
pcs holding -d 2025-03-06 -c USD
```

```console check
# Holding Report on 2025-03-06
  
  Total Portfolio Value: **$16,240.00**
  
  ## Securities
  
   Ticker | Quantity |   Price | Market Value 
  --------|----------|---------|--------------
   MSFT   |        5 | $420.00 |    $2,100.00 
  
  ## Cash
  
   Currency |    Balance |      Value 
  ----------|------------|------------
   EUR      | €10,000.00 | $11,000.00 
   USD      |  $3,200.00 |  $3,200.00 
  
  ## Counterparties
  
   Name       | Balance |   Value 
  ------------|---------|---------
   TaxAccount | -$60.00 | -$60.00
```