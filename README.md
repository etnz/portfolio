# Portfolio Management CLI

This is a command-line tool for managing your investment portfolio. It helps you track your securities, transactions, and analyze your portfolio's performance.

## Benefits

- **Comprehensive Portfolio Summary:** Generate detailed performance summaries, including daily, weekly, monthly, quarterly, yearly, and inception-to-date returns.
- **Time-Weighted Return (TWR) Calculation:** Uses the industry-standard TWR to accurately measure performance, removing the distorting effects of cash flows.
- **Multi-Currency Support:** Handle multiple currencies and convert them to a single reporting currency.
- **Transaction Validation:** Validate transactions before recording them to maintain data integrity.
- **Cost Basis Calculation:** Calculate the cost basis of your portfolio for tax purposes.

## Limitations

- **Data Source:** The tool relies on JSONL files for market data and transactions. It does not connect to live data sources.
- **No GUI:** This is a command-line only tool.
- **Manual Data Entry:** Transactions and market data must be entered manually.

## Getting Started

### Prerequisites

- [Go](https://golang.org/) installed on your system.
- make sure that `go install` will install binaries in your PATH. `go install` will install binaries in the directory named by the GOBIN environment variable, which defaults to $GOPATH/bin or $HOME/go/bin if the GOPATH
environment variable is not set.
- An API key for [EODHD](https://eodhd.com/) if you want to use the security search and update feature.

### Installation

To install (or update) the tool, simply use the following command:

```bash
go install github.com/etnz/portfolio/cmd/pcs@latest
```

This will compile and install the `pcs` command-line tool in your PATH.

Check the installation by running:

```bash
pcs help
```

```console
Usage: pcs <flags> <subcommand> <subcommand args>

Subcommands:
        commands         list all command names
        flags            describe all known top-level flags
        help             describe subcommands and their syntax

Subcommands for amundi:
        import-amundi    converts an Amundi transactions JSON file to JSONL format
        update-amundi    import transactions from an amundi jsonl file

Subcommands for analysis:
        gains            realized and unrealized gain analysis
        history          display asset value history
        holding          display detailed holdings for a specific date
        summary          display a portfolio performance summary

Subcommands for securities:
        add-security     add a new security to the market data
        import-investing  import public security prices from investing.com's CSV format
        search-security  search for securities using EODHD API
        update           update security prices from an external provider

Subcommands for transactions:
        buy              record the purchase of a security
        convert          converts cash from one currency to another within the portfolio
        declare          declare a new security
        deposit          record a cash deposit into the portfolio
        dividend         record a dividend payment for a security
        sell             record the sale of a security
        withdraw         record a cash withdrawal from the portfolio


Use "pcs flags" for a list of top-level flags
```

## About This Project

This project is an exercise in using AI to generate code. The maintainer is using Gemini Code Assist as a full stack software engineer:
* the initial ontology (naming concepts, like ledger), project structure, CLI commands names, consistency accross function names.
* All design documents, and implementation plans.
* This readme (except this section)
* Github issues interactions. Most of the comments, many issues descriptions.
* All commit messages.
* All the code, including tests.