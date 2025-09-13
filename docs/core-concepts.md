## Core Concepts

`pcs` is built on a few fundamental concepts that are essential to understanding how the tool works.

### The Ledger

The **Ledger** represents your portfolio. It's a chronological record of every transaction you make, from buying and selling securities to depositing cash and all the information you can get from the market (prices, splits, dividends, etc.). The ledger is stored in the `transactions.jsonl` file, and it serves as the single source of truth for all your financial activities.

### Counterparty Accounts

A **Counterparty Account** represents the financial balance with a specific external entity. It's used to track assets or liabilities that are not cash or securities, such as a loan to a friend, a tax liability, or an amount owed to a landlord. These are managed primarily with the `accrue` command, which can create a new account or update an existing one. A positive balance represents a receivable (an asset), while a negative balance represents a payable (a liability).

### The Security ID

The **Security ID** is a unique identifier for each asset in your portfolio. It's a powerful concept that allows `pcs` to handle a wide range of assets, from publicly traded stocks to private funds.

The ID is the crucial link between your **Ledger** and the **Market Data**. This separation allows you to use short, convenient tickers in your ledger (e.g., `AAPL`) without worrying about conflicts in a large market data store. The `declare` command creates this link, mapping your personal ticker to the globally unique Security ID. Market data provider (e.g eodhd) can fetch updates from the market solely based on the ID.

The ID is a rich format that has forms:

* **MSSI (Market-Specific Security Identifier)**: This is the standard for publicly traded securities. It's a combination of an ISIN ([International Securities Identification Number](https://en.wikipedia.org/wiki/International_Securities_Identification_Number)) and a MIC ([Market Identifier Code](https://en.wikipedia.org/wiki/Market_Identifier_Code)), separated by a period. For example, `US0378331005.XETR` represents Apple Inc. traded on the XETRA exchange.
* **CurrencyPair**: This is used for foreign exchange pairs. It's a six-character string created by concatenating two three-character ISO 4217 currency codes. For example, `EURUSD` represents the price of one Euro in terms of US Dollars.
* **ISIN Only**: This is used for funds as they are not traded on a specific exchange.
* **Private**: This is a generic, non-standard identifier for assets that don't have a public ID, such as a private equity investment or a corporate savings plan fund. A private ID must be at least 7 characters long and cannot contain a period and must not be interpretable as one of the above.
