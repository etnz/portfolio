# pcs: The Unified Portfolio Tracker - User Manual

Welcome to `pcs`, your private, local-first, command-line tool for gaining a clear, unified view of your entire investment portfolio. This manual will guide you through the core concepts, getting started, and the full range of commands available in `pcs`.

## Core Concepts

`pcs` is built on a few fundamental concepts that are essential to understanding how the tool works.

### The Ledger

The **Ledger** is the heart of your portfolio. It's a chronological record of every transaction you make, from buying and selling securities to depositing cash and receiving dividends. The ledger is stored in the `transactions.jsonl` file, and it serves as the single source of truth for all your financial activities.

### Market Data

The **Market Data** store contains all the information about the securities you track, including their definitions and price histories. This data is stored in the `market.jsonl` file and is used to value your holdings and calculate performance.

### The Security ID

The **Security ID** is a unique identifier for each asset in your portfolio. It's a powerful concept that allows `pcs` to handle a wide range of assets, from publicly traded stocks to private funds.

The ID is the crucial link between your **Ledger** and the **Market Data**. This separation allows you to use short, convenient tickers in your ledger (e.g., `AAPL`) without worrying about conflicts in a large market data store. The `declare` command creates this link, mapping your personal ticker to the globally unique Security ID.

The ID can take several forms:

* **MSSI (Market-Specific Security Identifier)**: This is the standard for publicly traded securities. It's a combination of an ISIN (International Securities Identification Number) and a MIC (Market Identifier Code), separated by a period. For example, `US0378331005.XETR` represents Apple Inc. traded on the XETRA exchange.
* **CurrencyPair**: This is used for foreign exchange pairs. It's a six-character string created by concatenating two three-character ISO 4217 currency codes. For example, `EURUSD` represents the price of one Euro in terms of US Dollars.
* **ISIN**: This is used for funds that are not traded on a specific exchange. It's a 12-character alphanumeric code that uniquely identifies a security.
* **Private**: This is a generic, non-standard identifier for assets that don't have a public ID, such as a private equity investment or a corporate savings plan fund. A private ID must be at least 7 characters long and cannot contain a period.

## Command Reference

```bash run
pcs help
```

```console check
Usage: pcs <flags> <subcommand> <subcommand args>

Subcommands:
        commands         list all command names
        flags            describe all known top-level flags
        help             describe subcommands and their syntax

Subcommands for amundi:
        import-amundi    converts an Amundi transactions JSON file to JSONL format
        update-amundi    import transactions from an amundi jsonl file

Subcommands for analysis:
        daily            display a daily portfolio performance report
        gains            realized and unrealized gain analysis
        history          display asset value history
        holding          display detailed holdings for a specific date
        summary          display a portfolio performance summary

Subcommands for securities:
        add-security     add a new security to the market data
        fetch-security   fetches and updates market data from external providers
        import-investing  import public security prices from investing.com's CSV format
        search-security  search for securities using EODHD API
        update-security  manually update a security's price or add a stock split

Subcommands for tools:
        format-ledger    formats the ledger file into a canonical form

Subcommands for transactions:
        accrue           record a non-cash transaction with a counterparty
        buy              record the purchase of a security
        convert          converts cash from one currency to another within the portfolio
        declare          declare a new security
        deposit          record a cash deposit into the portfolio
        dividend         record a dividend payment for a security
        sell             record the sale of a security
        withdraw         record a cash withdrawal from the portfolio


Use "pcs flags" for a list of top-level flags
```

## Advanced Concepts

### Cost Basis Methods

`pcs` supports two cost basis methods for calculating capital gains:

* **`average`**: This method calculates the cost basis by averaging the cost of all shares.
* **`fifo`**: This method (First-In, First-Out) assumes that the first shares purchased are the first ones sold.

You can specify the cost basis method using the `-method` flag on the `gains` command.

### Performance Calculation

`pcs` calculates portfolio performance using the **Time-Weighted Return (TWR)** method. TWR measures the compound growth rate of a portfolio, removing the distorting effects of cash flows. This is the industry standard for comparing investment manager performance.

### Extending `pcs`

`pcs` is designed to be extensible, allowing you to add your own custom commands. This is particularly useful for integrating with proprietary bank APIs, importing data from custom formats, or adding specialized analysis tools.

The primary and most flexible way to add functionality is by creating external command executables. When you run a command that `pcs` doesn't recognize (e.g., `pcs my-importer`), it will search your system's `PATH` for an executable file named `pcs-my-importer`.

If found, `pcs` will execute that file, passing along any additional arguments. This allows you to write extensions in any programming language (Go, Python, Bash, etc.) and keep them completely separate from the main `pcs` codebase.

**How it Works:**

1.  Create an executable script or binary (e.g., `pcs-my-importer`).
2.  Make sure it's executable (`chmod +x pcs-my-importer`).
3.  Place it in a directory that is part of your system's `$PATH`.
4.  You can now run it as if it were a native command: `pcs my-importer --some-flag`.

This is the recommended approach for adding custom importers or integrations.

#### Accessing Configuration from Extensions

To allow your external commands to seamlessly integrate with the user's setup, `pcs` passes its current configuration to the extension via environment variables. Your script or program can read these variables to know which files to access and which settings to use.

The following environment variables are set:

* `PCS_MARKET_FILE`: The absolute path to the `market.jsonl` file currently in use.
* `PCS_LEDGER_FILE`: The absolute path to the `transactions.jsonl` file currently in use.
* `PCS_DEFAULT_CURRENCY`: The default currency (e.g., "EUR") set by the user.

By reading these variables, your extension can operate on the same data as the core `pcs` tool without needing the user to specify file paths or API keys again.

#### Using Flexible Date Formats

To make entering dates faster and more intuitive, most commands that accept a date flag (like `-d` or `-end`) support several shorthand formats. You can use these formats instead of typing the full `YYYY-MM-DD` date.

The formats are interpreted in the following order:

1.  **Relative Duration Format**

    You can specify a date relative to today using a signed integer and a unit. The sign is mandatory.

    * **Format**: `[sign][number][unit]`
    * **Sign**: `+` for a future date, `-` for a past date.
    * **Unit**: `d` (days), `w` (weeks), `m` (months), `q` (quarters), `y` (years).

    | Example | Assuming Today is 2025-08-29 | Resulting Date |
    | :------ | :--------------------------- | :------------- |
    | `-1d`   | Yesterday                    | `2025-08-28`   |
    | `+1d`   | Tomorrow                     | `2025-08-30`   |
    | `+0d`   | Today                        | `2025-08-29`   |
    | `-2w`   | Two weeks ago                | `2025-08-15`   |
    | `-1m`   | One month ago                | `2025-07-29`   |

2.  **`[MM-]DD` Format**

    You can specify a day, or a month and a day, and the current year will be assumed. This format also has special handling for `0`.

    * **`DD`**: A day in the current month and year.
    * **`MM-DD`**: A specific month and day in the current year.
    * **`0` as the day**: Resolves to the last day of the *previous* month.
    * **`0` as the month**: Resolves to the corresponding day in December of the *previous* year.

    | Example | Assuming Today is 2025-08-29 | Resulting Date |
    | :------ | :--------------------------- | :------------- |
    | `27`    | The 27th of the current month | `2025-08-27`   |
    | `1-15`  | January 15th of current year | `2025-01-15`   |
    | `0`     | Last day of previous month   | `2025-07-31`   |
    | `8-0`   | Last day of July (month 8-1) | `2025-07-31`   |
    | `1-0`   | Last day of previous year    | `2024-12-31`   |
    | `0-15`  | Dec 15th of the previous year | `2024-12-15`   |
    | `0-0`   | Nov 30th of the previous year | `2024-11-30`   |

3.  **`YYYY-MM-DD` Format**

    If the input doesn't match any of the shorthand formats, `pcs` will try to parse it as a full standard date.

    | Example      | Resulting Date |
    | :----------- | :------------- |
    | `2024-02-29` | `2024-02-29`   |
