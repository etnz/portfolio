# Gains Report

**Command:** `pcs gains`

The `pcs gains` report answers the question: "What is the total economic change in my assets' value according to accounting principles?" It breaks down the change in value into realized (from sales) and unrealized (from price changes) gains and losses.

This report provides **essential clarity** on your investment performance and is crucial for understanding tax implications.

## Key Metrics and Their Calculation

*   **Realized Gain/Loss:** The profit or loss from selling a security. It's calculated by subtracting the cost basis of the sold asset from the proceeds of the sale. The report shows the total realized gain/loss for the specified period.
*   **Unrealized Gain/Loss:** The change in value of a security that you still own. It's the difference between the security's market value and its cost basis at the end of the period, minus the same calculation at the start of the period.
*   **Total Gain/Loss:** The sum of the realized and unrealized gains and losses for the period.

## Scenarios

### Baseline Gain/Loss

This scenario demonstrates the calculation of realized and unrealized gains for a security that has been partially sold.

```bash setup
# Fund the portfolio with EUR and USD.
pcs deposit -d 2025-01-01 -c EUR -a 10000
pcs deposit -d 2025-01-01 -c USD -a 5000
# Add stock to the ledger and make the first buy transaction.
pcs declare -d 2025-01-02 -s MSFT -id US0378331005.XNAS -c USD
pcs declare -d 2025-01-02 -s EURUSD -id EURUSD -c EUR
pcs buy -d 2025-01-02 -s MSFT -q 10 -a 4000
# Manually updating market data to explicitly show price changes.
# In a real-world daily routine, `pcs fetch eodhd` would automate this.
pcs price -s MSFT -d 2025-02-28 -p 400
pcs price -s MSFT -d 2025-03-05 -p 420
pcs price -s MSFT -d 2025-03-31 -p 450
pcs price -s EURUSD -d 2025-02-28 -p 1.1
pcs price -s EURUSD -d 2025-03-05 -p 1.1
pcs price -s EURUSD -d 2025-03-31 -p 1.1
# Sell the stock.
pcs sell -d 2025-03-06 -s MSFT -q 5 -a 2250
```

```bash run
pcs gains --period month -d 2025-03-31
```

```console check
# ledger Monthly Review for 2025-March

  *As of 2006-01-02 15:04:05*

   **Total Portfolio Value** | **€14,999.99** 
  ---------------------------|----------------
              Previous Value |     €14,545.45 
                             |                
                Capital Flow |                
              + Market Gains |       +€454.54 
               + Forex Gains |                
            **= Net Change** |    **€454.54** 
                             |                
                 Cash Change |     +€2,045.45 
     + Counterparties Change |                
       + Market Value Change |     -€1,590.90 
            **= Net Change** |    **€454.54** 
                             |                
                   Dividends |                
              + Market Gains |       +€454.54 
               + Forex Gains |                
            **=Total Gains** |   **+€454.54** 

  ## Tax View

   Asset     | Cost Basis (average) | Dividends |     Realized |   Unrealized 
  -----------|----------------------|-----------|--------------|--------------
   MSFT      |            $2,000.00 |           |     +$250.00 |     +$250.00 
   **Total** |        **€1,818.18** |     ** ** | **+€227.27** | **+€227.27**
```

### Impact of a Sale on Unrealized Gains

This scenario demonstrates a situation where an asset is bought before the reporting period and sold entirely within the period, resulting in a negative unrealized gain.

```bash setup
# Fund the portfolio with EUR and USD.
pcs deposit -d 2025-01-01 -c EUR -a 10000
pcs deposit -d 2025-01-01 -c USD -a 2000
# Add stock to the ledger and make the first buy transaction.
pcs declare -d 2025-01-01 -s GOOG -id US02079K3059.XNAS -c USD
pcs declare -d 2025-01-01 -s EURUSD -id EURUSD -c EUR
pcs buy -d 2025-01-02 -s GOOG -q 10 -a 1000
# Manually updating market data to explicitly show price changes. 
# In a real-world daily routine, `pcs fetch eodhd` would automate this.
pcs price -d 2025-02-28 -s GOOG -p 120
pcs price -d 2025-03-15 -s GOOG -p 110
pcs price -d 2025-03-31 -s GOOG -p 110
pcs price -d 2025-02-28 -s EURUSD -p 1.1
pcs price -d 2025-03-15 -s EURUSD -p 1.1
pcs price -d 2025-03-31 -s EURUSD -p 1.1
# Sell the stock.
pcs sell -d 2025-03-15 -s GOOG -q 10 -a 1100
```

```bash run
pcs gains --period month -d 2025-03-31
```

```console check
# ledger Monthly Review for 2025-March

  *As of 2006-01-02 15:04:05*

   **Total Portfolio Value** | **€11,909.08** 
  ---------------------------|----------------
              Previous Value |     €11,999.99 
                             |                
                Capital Flow |                
              + Market Gains |        -€90.90 
               + Forex Gains |                
            **= Net Change** |    **-€90.90** 
                             |                
                 Cash Change |       +€999.99 
     + Counterparties Change |                
       + Market Value Change |     -€1,090.90 
            **= Net Change** |    **-€90.90** 
                             |                
                   Dividends |                
              + Market Gains |        -€90.90 
               + Forex Gains |                
            **=Total Gains** |    **-€90.90** 

  ## Tax View

   Asset     | Cost Basis (average) | Dividends |    Realized | Unrealized 
  -----------|----------------------|-----------|-------------|------------
   GOOG      |                $0.00 |           |    +$100.00 |            
   **Total** |            **€0.00** |     ** ** | **+€90.90** |      ** **
```

**Explanation:**

The unrealized gain is negative because the security was sold within the reporting period. At the beginning of the period (2025-03-01), the security had an unrealized gain of 200 EUR (10 shares * (120 USD - 100 USD) / 1.1 EUR/USD). At the end of the period (2025-03-31), the position is closed, so the unrealized gain is 0. The change in unrealized gain for the period is therefore 0 - 200 = -200 EUR.