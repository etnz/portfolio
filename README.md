![pcs-large](docs/pcs-large.png)

[![Go Reference](https://pkg.go.dev/badge/github.com/etnz/portfolio.svg)](https://pkg.go.dev/github.com/etnz/portfolio)

# pcs: Your Private, Unified Portfolio Tracker

For the long-term investor with a scattered and diverse portfolio, `pcs` is the **durable** command-line tool that **unifies** your entire financial world.

By placing your financial data directly under your control in simple, local text files, `pcs` provides a permanent and auditable record of your wealth. It is **built to last**.

`pcs` provides **essential clarity and insights**, making it effortless to answer the two most critical questions at any point in your financial journey: "What is my net worth?" and "How is it performing?".


## The Solution: A Unified, Durable Portfolio

`pcs` solves this by being asset-agnostic and built for durability.
*   **Unify Everything**: It is designed from the ground up to track a diverse range of assets, from public stocks and private funds to liabilities, consolidating every part of your wealth into one coherent picture.
*   **Built to Last**: Your data lives in a simple, local text file (`transactions.jsonl`). You own it forever. The text-based format works perfectly with version control (like Git), creating an immutable, auditable history of your wealth that will outlast any proprietary cloud service.
*   **Essential Clarity**: Financial tools often overwhelm users with complex analytics. `pcs` focuses on providing simple, clear answers to the core questions that matter most.

## Getting Started: Your First Unified Portfolio

This tutorial will walk you through the process of setting up your portfolio and tracking your first investment.

### Installation

To get started, ensure you have Go installed on your system. Then, you can install `pcs` with a single command:

```bash
go install github.com/etnz/portfolio/cmd/pcs@latest
```

### Declaring Your Assets

Before you can track an asset, you need to know exactly what asset you want to track.
For instance Apple's stocks can be exchanged in USD on the Nasdaq, but also as EUR
on the XETRA exchange.

Let's declare a public stock (Apple) and a private fund in your corporate savings plan.
declaring is given them a short mnemonic name to include all that identifying information
about the stock.


```bash run
pcs declare -d 2025-08-27 -s AAPL -id US0378331005.XETR -c EUR
```

```console check
✅ Successfully recorded transaction in ledger "ledger".
```

> [!NOTE]
> id uses Apple stock's ISIN followed by the exchange MIC code (XETR for XETRA). You can find this information on any financial websites. `pcs search Apple` can also help you find it.


Your corporate savings plan let you buy shares of funds that unfortunately are often not publicly traded. You can still track it by giving it a unique "private" identifier, and updating them manually (or by other tricks):

```bash run
pcs declare -d 2025-08-27 -s BankFund1 -id My-bank-Fund1 -c EUR
```

```console check
✅ Successfully recorded transaction in ledger "ledger".
```

> [!NOTE]
> The `-id` is a private identifier that uniquely represents your bank's private fund.

Publicly traded securities can be fetched automatically. `pcs` supports `eodhd` out of the box, but its extension mechanism allows you to use your preferred data provider.

Privately traded securities, often found in bank savings accounts, can be more tedious to update. You can either update the fund price manually using the CLI or automate the process by writing or finding a suitable provider extension.


### Recording Transactions

Let's record that we have deposited some cash into your account.

```bash run
pcs deposit -d 2025-08-27 -a 10000 -c EUR
```

```console check
✅ Successfully recorded transaction in ledger "ledger".
```

 [!NOTE]
> In the whole documentation we always explicitly set the date for clarity, but the defaut date is usually correct.



Let's record that we bought some Apple's stock with that money.

```bash run
pcs buy -d 2025-08-27 -s AAPL -q 10 -a 1500.0
```

```console check
✅ Successfully recorded transaction in ledger "ledger".
```

Let's record a buy in the corporate savings plan.

```bash run
pcs buy -d 2025-08-27 -s BankFund1 -q 100 -a 1200.0
```

```console check
✅ Successfully recorded transaction in ledger "ledger".
```


### Keeping Your Portfolio Up-to-Date

Recording the transaction you have initiated is not enough to compute the value of your portfolio. You also need the latest information from the market.
`pcs` can automatically fetch market data from providers. Each provider has its own command, for example:

```bash
pcs eodhd fetch
pcs insee fetch
```

You can also manually set the price for any asset using the `pcs price` command.

However for the purpose of the tutorial, we are only going to use the manual method. We can read from any financial site Apple's closing price on 2025-08-27:

```bash run
pcs price -s AAPL -d 2025-08-27 -p 193.20
```

```console check
✅ Successfully recorded transaction in ledger "ledger".
```

From your corporate saving bank site you can get the price on 2025-08-27:

```bash run
pcs price -s BankFund1 -d 2025-08-27 -p 11.23
```

```console check
✅ Successfully recorded transaction in ledger "ledger".
```


### The Payoff: Reporting

Now, you can see a unified view of your portfolio: 

```bash run
pcs review -d 2025-08-27
```

```console check
# ledger Review for 2025-08-27

  *As of 2006-01-02 15:04:05*

   **Total Portfolio Value** | **€10,355.00** 
  ---------------------------|----------------
              Previous Value |          €0.00 
                             |                
                Capital Flow |    +€10,000.00 
              + Market Gains |       +€355.00 
               + Forex Gains |              - 
            **= Net Change** | **€10,355.00** 
                             |                
                 Cash Change |     +€7,300.00 
     + Counterparties Change |              - 
       + Market Value Change |     +€3,055.00 
            **= Net Change** | **€10,355.00** 
                             |                
                   Dividends |              - 
              + Market Gains |       +€355.00 
               + Forex Gains |              - 
            **=Total Gains** |   **+€355.00** 

  ## Accounts

   **Cash Accounts** |         Value | Forex % 
  -------------------|---------------|---------
                 EUR |     €7,300.00 |       - 
           **Total** | **€7,300.00** |         

   **Counterparty Accounts** | Value 
  ---------------------------|-------
                   **Total** | **-** 

  ## Consolidated Asset Report

   Asset     | Start Value |     End Value |   Trading Flow |  Market Gain | Realized Gain | Unrealized Gain | Dividends |       TWR 
  -----------|-------------|---------------|----------------|--------------|---------------|-----------------|-----------|-----------
   AAPL      |        0.00 |     €1,932.00 |     +€1,500.00 |     +€432.00 |             - |        +€432.00 |         - |         - 
   BankFund1 |        0.00 |     €1,123.00 |     +€1,200.00 |      -€77.00 |             - |         -€77.00 |         - |    -0.00% 
   **Total** |   **€0.00** | **€3,055.00** | **+€2,700.00** | **+€355.00** |         **-** |    **+€355.00** |     **-** | **+NaN%** 

  ## Transactions

  • 2025-08-27: Declare "AAPL" as "US0378331005.XETR" in EUR
  •           : Declare "BankFund1" as "My-bank-Fund1" in EUR
  •           : Update price for "AAPL"=193.2000
  •           : Update price for "BankFund1"=11.2300
  •           : Deposit €10,000.00
  •           : Buy 10 of "AAPL" for €1,500.00
  •           : Buy 100 of "BankFund1" for €1,200.00
```


## User Manual

For a detailed guide on all the features and commands, please refer to the [User Manual](./docs/readme.md).