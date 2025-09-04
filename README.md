[![Go Reference](https://pkg.go.dev/badge/github.com/etnz/portfolio.svg)](https://pkg.go.dev/github.com/etnz/portfolio)

# pcs: Your Private, Unified Portfolio Tracker

In a world where your investments are scattered across multiple platforms—from corporate savings plans and retirement accounts to various online brokers—getting a single, clear view of your financial health is a challenge. `pcs` is a local-first, command-line tool designed to solve this problem by providing a unified, private, and auditable view of your entire investment portfolio.

## The Challenge: A Scattered Portfolio

Many of us have assets spread across a variety of disconnected accounts:

*   **Corporate Savings Plans:** Often managed by specific institutions and not easily tracked in standard brokerage accounts.
*   **Retirement Plans:** Typically held in separate, dedicated accounts.
*   **Corporate Stock Options:** Managed by a specific broker, adding another silo to your portfolio.
*   **Online Trading Platforms:** Modern online banks and trading platforms that offer stock and crypto trading.
*   **Traditional Life Insurance:** Held in a traditional bank, with its own interface and reporting.

This scattered landscape makes it incredibly difficult to answer a simple question: "What is my total net worth, and how is it performing?"

## The Solution: A Unified, Private Portfolio

`pcs` allows you to bring all your assets into a single view, giving you a clear picture of your entire portfolio. The tool is designed to handle both publicly-traded stocks and private, hard-to-track assets, like corporate savings plans or even real estate. Because `pcs` is a local-first tool that operates on your own machine, you have complete privacy and control over your financial data.

## Getting Started: Your First Unified Portfolio

This tutorial will walk you through the process of setting up your portfolio and tracking your first investment.

### Installation

To get started, ensure you have the following prerequisites:
- [Go](https://golang.org/) is installed on your system.
- [EODHD](https://eodhd.com/) API key must be set in the `EODHD_API_KEY` environment variable (free tier is sufficient) or you can pass it as a flag to the `pcs` command.

Then, you can install `pcs` with a single command:

```bash
go install github.com/etnz/portfolio/cmd/pcs@latest
```

### Declaring Your Assets in the Market Data

Before you can track an asset, it needs to be declared in the `market.jsonl` file. 

Let's declare a public stock (Apple) and a private fund in your corporate savings plan.


```bash run
pcs add-security -s AAPL -id US0378331005.XETR -c EUR
```

```console check
✅ Successfully added security 'AAPL' to the market data.
```

> [!NOTE]
> id uses Apple stock's ISIN followed by the exchange MIC code (XETR for XETRA). You can find this information on any financial websites. `pcs search-security Apple` can also help you find it.


Your corporate savings plan let you buy shares of funds that unfortunately are publicly traded. You can still track it by giving it a unique identifier, and updating them manually:

```bash run
pcs add-security -s BankFund1 -id My-bank-Fund1 -c EUR
```

```console check
✅ Successfully added security 'BankFund1' to the market data.
```

> [!NOTE]
> -id is private identifier that identifies your bank's private fund.

Public securities will have their prices fetched automatically, while private securities will need to be updated manually.

### Declaring Your Assets in your Ledger

Also it might look redundant, but you need to declare your assets in your ledger as well. This is because you might have multiple ledgers holding the same security, and you want to track them separately.

```bash run
pcs declare -d 2025-08-27 -s AAPL -id US0378331005.XETR -c EUR
```

```console check
Successfully appended transaction to transactions.jsonl
```

```bash run
pcs declare -d 2025-08-27 -s BankFund1 -id My-bank-Fund1 -c EUR
```

```console check
Successfully appended transaction to transactions.jsonl
```

### Recording Transactions

Let's deposit some cash into your account.

```bash run
pcs deposit -d 2025-08-27 -a 10000 -c EUR
```

```console check
Successfully appended transaction to transactions.jsonl
```

Let's buy some Apple stock.

```bash run
pcs buy -d 2025-08-27 -s AAPL -q 10 -a 1500.0
```

```console check
Successfully appended transaction to transactions.jsonl
```

Let's record a buy in the corporate savings plan.

```bash run
pcs buy -d 2025-08-27 -s BankFund1 -q 100 -a 1200.0
```

```console check
Successfully appended transaction to transactions.jsonl
```


### Keeping Your Portfolio Up-to-Date

You can update the prices for your securities using the `update-security` command.

For publicly traded securities or assets you can get the latest prices automatically, which is very handy for daily updates. You would just run `pcs update-security` without any flags.

However for the purpose of this tutorial only, let's manually set the price for Apple stock to its closing price on 2025-08-27:

```bash run
pcs update-security -id US0378331005.XETR -d 2025-08-27 -p 193.20
```

```console check
Successfully set price for US0378331005.XETR on 2025-08-27 to 193.20.
```

For private assets, you have to manually update their price using the same command. Or write your own command to fetch prices from your bank's API if they provide one. Here we'll set the price for your corporate savings plan fund to its value on 2025-08-27:

```bash run
pcs update-security -id My-bank-Fund1 -d 2025-08-27 -p 11.23
```

```console check
Successfully set price for My-bank-Fund1 on 2025-08-27 to 11.23.
```

> [!NOTE]
> The `pcs update-security` command, when run without any flags, will automatically fetch the latest prices for all publicly traded securities from an external provider (requires EODHD API key). This is usually run daily to get yesterday's closing prices.



### The Payoff: Reporting

Now, you can see a unified view of your portfolio:

```bash run
pcs holding -d 2025-08-27
```

```console check
# Holding Report on 2025-08-27
  
  Total Portfolio Value: **€10,355.00**
  
  ## Securities
  
   Ticker    | Quantity |   Price | Market Value 
  -----------|----------|---------|--------------
   AAPL      |       10 | €193.20 |    €1,932.00 
   BankFund1 |      100 |  €11.23 |    €1,123.00 
  
  ## Cash
  
   Currency |   Balance |     Value 
  ----------|-----------|-----------
   EUR      | €7,300.00 | €7,300.00
```


And gains on the portfolio:

```bash run
pcs gains -period=day -d 2025-08-27
```

```console check
Capital Gains Report (Method: average) for 2025-08-27 to 2025-08-27 (in EUR)
--------------------------------------------------------------------------------
Security               Realized Gain/Loss Unrealized Gain/Loss      Total Gain/Loss
--------------------------------------------------------------------------------
AAPL                                 0.00               432.00               432.00
BankFund1                            0.00               -77.00               -77.00
--------------------------------------------------------------------------------
Total                                0.00               355.00               355.00
```

## User Manual

For a detailed guide on all the features and commands, please refer to the [User Manual](./docs/UserManual.md).

## About This Project

This project is an exercise in using AI to generate code. The maintainer is using Gemini Code Assist as a full stack software engineer:
* the initial ontology (naming concepts, like ledger), project structure, CLI commands names, consistency accross function names.
* All design documents, and implementation plans.
* This readme (except this section)
* Github issues interactions. Most of the comments, many issues descriptions.
* All commit messages.
* All the code, including tests.