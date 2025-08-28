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

Here is a complete reference for all the commands available in `pcs`.

### Securities Management

* **`add-security`**: Adds a new security to the market data file.
    * `-s`: The ticker symbol for the security.
    * `-id`: The unique identifier for the security.
    * `-c`: The 3-letter currency code for the security.
    * `-from-ledger`: Adds all securities declared in the ledger file.
* **`search-security`**: Searches for securities using the EODHD API.
    * `-show-errors`: Displays entries with invalid ISINs.
* **`update-security`**: Updates security prices.
    * `-start`: The start date for automatic updates.
    * `-end`: The end date for automatic updates.
    * `-id`: The security ID for manual updates.
    * `-p`: The price for manual updates.
    * `-d`: The date for manual updates.
* **`import-investing`**: Imports security prices from an investing.com CSV file.
    * `-file`: The input file path.

### Transaction Management

* **`buy`**: Records the purchase of a security.
    * `-d`: Transaction date (YYYY-MM-DD).
    * `-s`: Security ticker.
    * `-q`: Number of shares.
    * `-a`: Total amount paid for the shares.
    * `-m`: An optional memo for the transaction.
* **`sell`**: Records the sale of a security.
    * `-d`: Transaction date (YYYY-MM-DD).
    * `-s`: Security ticker.
    * `-a`: Total amount received for the shares.
    * `-q`: Number of shares (if omitted, all shares are sold).
    * `-m`: An optional memo for the transaction.
* **`dividend`**: Records a dividend payment.
    * `-d`: Transaction date (YYYY-MM-DD).
    * `-s`: Security ticker receiving the dividend.
    * `-a`: Total dividend amount received.
    * `-m`: An optional memo.
* **`deposit`**: Records a cash deposit.
    * `-d`: Transaction date (YYYY-MM-DD).
    * `-a`: Amount of cash to deposit.
    * `-c`: Currency of the deposit.
    * `-m`: An optional memo.
* **`withdraw`**: Records a cash withdrawal.
    * `-d`: Transaction date (YYYY-MM-DD).
    * `-a`: Amount of cash to withdraw.
    * `-c`: Currency of the withdrawal.
    * `-m`: An optional memo.
* **`convert`**: Converts cash from one currency to another.
    * `-d`: Transaction date (YYYY-MM-DD).
    * `-fc`: Source currency code.
    * `-fa`: Amount of cash to convert from the source currency.
    * `-tc`: Destination currency code.
    * `-ta`: Amount of cash received in the destination currency.
    * `-m`: An optional memo.
* **`declare`**: Declares a new security in the ledger.
    * `-s`: The ledger-internal ticker to define.
    * `-id`: The full, unique security ID.
    * `-c`: The currency of the security.
    * `-d`: Transaction date (YYYY-MM-DD).
    * `-m`: An optional memo.

### Analysis and Reporting

* **`summary`**: Displays a portfolio performance summary.
    * `-d`: The date for the summary.
    * `-c`: The reporting currency for the summary.
    * `-u`: Updates with the latest intraday prices before calculating the summary.
* **`holding`**: Displays detailed holdings for a specific date.
    * `-d`: The date for the holdings report.
    * `-c`: The reporting currency for market values.
    * `-u`: Updates with the latest intraday prices before calculating the report.
* **`history`**: Displays the value of an asset or cash account over time.
    * `-s`: The security ticker to report on.
    * `-c`: The currency of the cash account to report on.
* **`gains`**: Provides realized and unrealized gain analysis.
    * `-period`: A predefined period (day, week, month, quarter, year).
    * `-start`: The start date of the reporting period.
    * `-end`: The end date of the reporting period.
    * `-c`: The reporting currency.
    * `-method`: The cost basis method (average, fifo).
    * `-u`: Updates with the latest intraday prices before calculating gains.

### Amundi Integration

* **`import-amundi`**: Converts an Amundi transactions JSON file to the standard JSONL format. This command takes the file path as a positional argument.
* **`update-amundi`**: Updates security prices from the Amundi portal.
    * `-start`: The start date (YYYY-MM-DD).
    * `-H`: Passes headers to run the URI (for authentication).

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