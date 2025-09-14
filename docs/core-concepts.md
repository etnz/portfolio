## Core Concepts

`pcs` is built on a few fundamental concepts that are essential to understanding how the tool works.

### The Ledger

The **Ledger** is the heart of your financial record, embodying the pillar of **durability**. It's a simple, human-readable text file (`transactions.jsonl`) that contains a chronological record of every transaction you make. Because you own this file, your financial history is permanent and safe from the whims of any cloud provider. It serves as the single, auditable source of truth for your entire portfolio.

### Counterparty Accounts

A **Counterparty Account** is a key feature for **unifying** your complete financial picture. It represents the financial balance with any external entity, allowing you to track assets and liabilities that aren't traditional securities. This includes loans to friends, tax liabilities, or rent owed. By tracking these, `pcs` ensures that your net worth calculation is truly comprehensive.

### The Security ID

The **Security ID** is the mechanism that enables `pcs` to **unify** a diverse range of assets. It's a unique, unambiguous identifier for everything you own, from publicly traded stocks (using standard ISINs) to private funds in a corporate savings plan.

The ID is the crucial link between your **Ledger** and the **Market Data**. This separation allows you to use short, convenient tickers in your ledger (e.g., `AAPL`) without worrying about conflicts in a large market data store. The `declare` command creates this link, mapping your personal ticker to the globally unique Security ID. Market data provider (e.g eodhd) can fetch updates from the market solely based on the ID.

The ID is a rich format that has forms:

* **MSSI (Market-Specific Security Identifier)**: This is the standard for publicly traded securities. It's a combination of an ISIN ([International Securities Identification Number](https://en.wikipedia.org/wiki/International_Securities_Identification_Number)) and a MIC ([Market Identifier Code](https://en.wikipedia.org/wiki/Market_Identifier_Code)), separated by a period. For example, `US0378331005.XETR` represents Apple Inc. traded on the XETRA exchange.
* **CurrencyPair**: This is used for foreign exchange pairs. It's a six-character string created by concatenating two three-character ISO 4217 currency codes. For example, `EURUSD` represents the price of one Euro in terms of US Dollars.
* **ISIN Only**: This is used for funds as they are not traded on a specific exchange.
* **Private**: This is a generic, non-standard identifier for assets that don't have a public ID, such as a private equity investment or a corporate savings plan fund. A private ID must be at least 7 characters long and cannot contain a period and must not be interpretable as one of the above.
