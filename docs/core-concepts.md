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
