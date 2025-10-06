# `pcs`: A Deep Dive for the Financially Savvy

This document provides a detailed overview of the `pcs` portfolio tracking tool, designed for users with a solid understanding of financial concepts. It will break down the core components of the `pcs` system and then delve into the specifics of each command, providing practical, real-world examples.

## Core Concepts

The `pcs` tool is built upon a few key concepts that are essential to understanding its functionality.

#### The Ledger

Within `pcs`, the **Ledger** is the immutable, append-only journal of all financial events. Stored locally as `transactions.jsonl`, it represents the single source of truth for the portfolio's entire history. It is not just a record of user-initiated trades but also incorporates market data events (e.g., price updates, splits, dividends), making it a self-contained and fully auditable dataset. This local-first, text-based architecture ensures data sovereignty and durability, allowing it to be version-controlled with tools like Git. All portfolio states and metrics are derived by processing this ledger up to a specific point in time.

#### Accrual Basis of Accounting

`pcs` operates on the **accrual basis of accounting** to provide a more accurate measure of economic reality. This principle dictates that your net worth changes the moment you gain a legal right to an asset or incur an obligation, not just when cash changes hands. For example, your net worth increases when you issue an invoice (creating a receivable), not weeks later when the client pays it. Similarly, your net worth decreases the moment you realize a capital gain, as a tax liability is instantly created, even if the tax is paid months later.

This method provides a truer, real-time picture of your financial position. The `accrue` command is the primary tool for recording these non-cash events, which are then settled using `deposit` or `withdraw` transactions.

#### Counterparty Accounts

A **Counterparty Account** is the core mechanism that enables `pcs` to implement the accrual basis of accounting. It is an abstraction for tracking off-balance-sheet assets and liabilities, representing a financial relationship with any external entity.

This mechanism is used to model:

* **Receivables**: Amounts owed to the portfolio (e.g., loans made, accrued income). These are positive balances.
* **Payables**: Amounts owed by the portfolio (e.g., accrued tax liabilities, loans taken). These are negative balances.

By using the `accrue` command to create these balances, you can ensure your net worth is always up-to-date. Subsequent `deposit` or `withdraw` transactions using the `-settles` flag are then treated as internal transfers that convert a receivable into cash or settle a payable with cash.

#### The Security ID

The **Security ID** is the canonical, unambiguous identifier for any asset tracked within `pcs`. It decouples the user's short-hand, portfolio-specific `ticker` from a globally unique identifier. This is critical for fetching accurate market data from external providers and preventing symbol collisions. The `declare` command establishes this mapping.

The supported ID formats are:

* **MSSI (Market-Specific Security Identifier)**: The standard for publicly traded securities, combining an **ISIN** (ISO 6166) and a **MIC** (ISO 10383) (e.g., `US0378331005.XNAS` for Apple on NASDAQ). This provides precision for multi-listed equities.
* **CurrencyPair**: A six-character concatenation of two ISO 4217 codes for foreign exchange pairs (e.g., `EURUSD`).
* **ISIN Only**: Used for assets like mutual funds that are not traded on a specific exchange.
* **Private**: A user-defined string for non-standard assets (e.g., private equity, real estate), which must not be parsable as any of the other standard formats.

#### Cash Account

In `pcs`, a **Cash Account** is not an explicitly declared entity but an implicit balance derived from the transaction history for a given currency. The balance for any currency at any point in time is the sum of all cash-affecting transactions (`deposit`, `withdraw`, `buy`, `sell`, `convert`) up to that date. This allows for multi-currency cash management without the need for manual account setup.

#### Cash Flow

**Cash Flow** is a critical performance metric. However, the term as used in `pcs` reports can be misleading if taken literally; it should be interpreted as the **external flow of capital** into or out of the portfolio. This distinction is crucial for accurate performance calculation (like Time-Weighted Return) and is a direct consequence of the accrual accounting method.

* `deposit` and `withdraw` transactions are treated as external capital flows by default.
* A `dividend` transaction records income, but the cash is considered external to the portfolio (e.g., paid to a separate bank account). Therefore, the `dividend` itself does **not** generate a cash flow event within the portfolio. A subsequent `deposit` is required to represent the cash entering the portfolio, and this is treated as a standard external capital inflow.
* When a `deposit` or `withdraw` includes the `-settles` flag to interact with a Counterparty Account, it is considered an **internal transfer** and does not impact the external capital flow.
* An `accrue` transaction is a non-cash event but is treated as an external capital flow because it represents a change in the portfolio's total economic value.

#### Market Gains

**Market Gains** represent the component of an asset's return attributable solely to price movement, stripping out the impact of capital flows (buys and sells). It is calculated as the change in total market value over a period, less the net trading flow during that same period. This provides a pure measure of an asset's performance in the market.

#### Realized vs. Unrealized Gains

`pcs` distinguishes between two types of capital gains for accurate tax and performance reporting:

* **Realized Gains**: The profit or loss that is "locked in" upon the sale of an asset. It is calculated as the sale proceeds minus the asset's `Cost Basis`.
* **Unrealized Gains**: The "on-paper" profit or loss of an asset still held in the portfolio. It is the difference between the asset's current market value and its `Cost Basis`.

#### Cost Basis & Cost Basis Method

The **Cost Basis** is the original value of an asset for tax purposes, derived from its purchase history. When multiple lots of a security are acquired at different prices, the **Cost Basis Method** determines how the cost of sold shares is calculated. `pcs` supports two primary methods:

* `fifo` (First-In, First-Out): Assumes the first shares purchased are the first ones sold.
* `average`: Uses the weighted average cost of all shares held at the time of sale.

#### Stock Price: Raw vs. Adjusted

`pcs` stores **raw (unadjusted) prices** in the ledger. Corporate actions that affect price, such as splits and dividends, are recorded as distinct transactions. This architectural choice ensures that the ledger is a faithful and auditable record of all historical events. The calculation engine internally accounts for these events when computing metrics, ensuring that performance and cost basis calculations are accurate without altering historical price data.

#### Stock Split

A **Stock Split** is recorded via a `split` transaction. It adjusts the quantity of shares held for a security by a specified ratio (`-num`/`-den`). This operation proportionally adjusts the cost-basis-per-share for all existing lots of the security, ensuring that the total cost basis of the position remains unchanged while accurately reflecting the new number of shares.

#### Dividend Payment

A `dividend` transaction records the dividend amount *per share*. The system then calculates the total dividend income based on the number of shares held on the transaction date. This income is a component of the portfolio's total return. The cash from the dividend is not automatically added to a cash account; it is assumed to be an external event until a corresponding `deposit` transaction is explicitly recorded, allowing for accurate tracking of dividends that may be paid to an external bank account.

> [!IMPORTANT]
> `pcs` records the dividend in the currency it was actually paid in, which may differ from the security's trading currency. For example, a US-domiciled stock traded in EUR on a European exchange will still pay its dividend in USD. This ensures that multi-currency income is tracked accurately. When fetching data automatically, the currency is taken directly from the provider; when adding a dividend manually via the CLI, the currency can be specified with the `-c` flag or it defaults to the security's declared currency.

### Commands and Flags

The following is a comprehensive breakdown of each `pcs` command used to record transactions in the ledger.

#### `accrue`

Recognizes an off-balance-sheet asset (receivable) or liability (payable) with a counterparty.

* **Flags**:
    * `-d`: (Optional) Transaction date. Defaults to the current day.
    * `-payable`: (Required\*) The counterparty account to which the user owes money.
    * `-receivable`: (Required\*) The counterparty account that owes money to the user.
    * `-a`: (Required) Amount of cash to accrue.
    * `-c`: (Optional) Currency of the accrual. Defaults to "EUR".
    * `-m`: (Optional) A descriptive memo for the transaction.
    * *\*Either `-payable` or `-receivable` must be specified, but not both.*

1.  **Accruing capital gains tax liability after a sale**:
    ```bash demo
    pcs init -d 2025-01-01 -c EUR
    pcs accrue -d 2025-12-31 -payable TaxAuthority -a 15200 -c EUR -m "Estimated CGT for FY2025"
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    A new counterparty account 'TaxAuthority' has been created.
    
    
      • 2025-01-01: init
      • 2025-12-31: Accrue payable €15,200.00 to "TaxAuthority"
    ```

2.  **Recording interest receivable on a private loan**:
    ```bash demo
    pcs init -d 2025-01-01 -c USD
    pcs accrue -d 2025-06-30 -receivable PrivateLoan_JohnDoe -a 250 -c USD -m "H1 2025 interest"
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    A new counterparty account 'PrivateLoan_JohnDoe' has been created.
    
    
      • 2025-01-01: init
      • 2025-06-30: Accrue receivable $250.00 from "PrivateLoan_JohnDoe"
    ```

3.  **Recognizing accrued but unpaid salary/bonus**:
    ```bash demo
    pcs init -d 2025-01-01 -c EUR
    pcs accrue -d 2025-12-31 -receivable EmployerCorp -a 10000 -c EUR -m "FY2025 performance bonus"
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    A new counterparty account 'EmployerCorp' has been created.
    
    
      • 2025-01-01: init
      • 2025-12-31: Accrue receivable €10,000.00 from "EmployerCorp"
    ```

4.  **Creating a payable for a professional service invoice received**:
    ```bash demo
    pcs init -d 2025-01-01 -c USD
    pcs accrue -d 2025-11-30 -payable LegalServicesLLC -a 5000 -c USD -m "Invoice #INV-2025-11-LS"
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    A new counterparty account 'LegalServicesLLC' has been created.
    
    
      • 2025-01-01: init
      • 2025-11-30: Accrue payable $5,000.00 to "LegalServicesLLC"
    ```

5.  **Marking-to-market a forward contract receivable**:
    ```bash demo
    pcs init -d 2025-01-01 -c CHF
    pcs accrue -d 2025-09-30 -receivable FwdContract_XYZ -a 1250 -c CHF -m "Q3 MTM adjustment"
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    A new counterparty account 'FwdContract_XYZ' has been created.
    
    
      • 2025-01-01: init
      • 2025-09-30: Accrue receivable 1,250.00 CHF from "FwdContract_XYZ"
    ```

#### `buy`

Records the acquisition of a security, establishing a new cost basis lot and debiting the corresponding cash account.

* **Flags**:
    * `-d`: (Optional) Transaction date. Defaults to the current day.
    * `-s`: (Required) Security ticker.
    * `-q`: (Required) Number of shares purchased.
    * `-a`: (Required) Total amount paid for the shares.
    * `-m`: (Optional) A descriptive memo for the transaction.

1.  **Acquiring a block of ETFs**:
    ```bash demo
    pcs init -d 2025-01-01 -c USD
    pcs declare -d 2025-01-01 -s VTI -id US9229087690.XNAS -c USD
    pcs deposit -d 2025-01-01 -a 30000 -c USD
    pcs buy -d 2025-03-10 -s VTI -q 100 -a 21550.75 -m "Quarterly portfolio rebalance"
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Declare "VTI" as "US9229087690.XNAS" in USD
      •           : Deposit $30,000.00
      • 2025-03-10: Buy 100 of "VTI" for $21,550.75
    ```

2.  **Dollar-cost averaging into a mutual fund**:
    ```bash demo
    pcs init -d 2025-01-01 -c USD
    pcs declare -d 2025-01-01 -s FXAIX -id US3159117502.XNAS -c USD
    pcs deposit -d 2025-01-01 -a 5000 -c USD
    pcs buy -d 2025-02-01 -s FXAIX -q 31.25 -a 5000
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Declare "FXAIX" as "US3159117502.XNAS" in USD
      •           : Deposit $5,000.00
      • 2025-02-01: Buy 31.25 of "FXAIX" for $5,000.00
    ```

3.  **Exercising employee stock options (cost is strike price)**:
    ```bash demo
    pcs init -d 2025-01-01 -c USD
    pcs declare -d 2025-01-01 -s ADBE -id US00724F1012.XNAS -c USD
    pcs deposit -d 2025-01-01 -a 10000 -c USD
    pcs buy -d 2025-11-15 -s ADBE -q 500 -a 5000.00 -m "NESO exercise, strike @ $10.00"
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Declare "ADBE" as "US00724F1012.XNAS" in USD
      •           : Deposit $10,000.00
      • 2025-11-15: Buy 500 of "ADBE" for $5,000.00
    ```

4.  **Buying into a private placement**:
    ```bash demo
    pcs init -d 2025-01-01 -c USD
    pcs declare -d 2025-01-01 -s PrivateStartupX -id "Private Startup X Equity" -c USD
    pcs deposit -d 2025-01-01 -a 25000 -c USD
    pcs buy -d 2025-06-01 -s PrivateStartupX -q 10000 -a 25000 -m "Series A round"
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Declare "PrivateStartupX" as "Private Startup X Equity" in USD
      •           : Deposit $25,000.00
      • 2025-06-01: Buy 10000 of "PrivateStartupX" for $25,000.00
    ```

5.  **Acquiring a foreign stock, specifying total cost in local currency**:
    ```bash demo
    pcs init -d 2025-01-01 -c EUR
    pcs declare -d 2025-01-01 -s SIE.XETR -id DE0007236101.XETR -c EUR
    pcs deposit -d 2025-01-01 -a 10000 -c EUR
    pcs buy -d 2025-04-05 -s SIE.XETR -q 50 -a 7525.50
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Declare "SIE.XETR" as "DE0007236101.XETR" in EUR
      •           : Deposit €10,000.00
      • 2025-04-05: Buy 50 of "SIE.XETR" for €7,525.50
    ```

#### `convert`

Executes a foreign exchange transaction between two internal cash accounts.

* **Flags**:
    * `-d`: (Optional) Transaction date. Defaults to the current day.
    * `-fc`: (Required) Source currency code.
    * `-fa`: (Required) Amount of cash to convert from the source currency.
    * `-tc`: (Required) Destination currency code.
    * `-ta`: (Required) Amount of cash received in the destination currency.
    * `-m`: (Optional) A descriptive memo for the transaction.

1.  **Spot conversion for an international purchase**:
    ```bash demo
    pcs init -d 2025-01-01 -c EUR
    pcs deposit -d 2025-01-01 -a 10000 -c USD
    pcs convert -d 2025-04-15 -fc USD -fa 10000 -tc EUR -ta 9250.50 -m "FX for asset purchase"
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Deposit $10,000.00
      • 2025-04-15: Convert $10,000.00 to €9,250.50
    ```

2.  **Repatriating dividends received in a foreign currency**:
    ```bash demo
    pcs init -d 2025-01-01 -c EUR
    pcs deposit -d 2025-01-01 -a 520.25 -c GBP
    pcs convert -d 2025-07-02 -fc GBP -fa 520.25 -tc EUR -ta 610.90
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Deposit £520.25
      • 2025-07-02: Convert £520.25 to €610.90
    ```

3.  **Funding a foreign currency account ahead of a trip**:
    ```bash demo
    pcs init -d 2025-01-01 -c EUR
    pcs deposit -d 2025-01-01 -a 2000 -c EUR
    pcs convert -d 2025-08-20 -fc EUR -fa 2000 -tc JPY -ta 295000
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Deposit €2,000.00
      • 2025-08-20: Convert €2,000.00 to ¥295,000
    ```

4.  **Closing a small residual foreign cash balance**:
    ```bash demo
    pcs init -d 2025-01-01 -c EUR
    pcs deposit -d 2025-01-01 -a 125.50 -c CHF
    pcs convert -d 2025-12-30 -fc CHF -fa 125.50 -tc EUR -ta 128.20
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Deposit 125.50 CHF
      • 2025-12-30: Convert 125.50 CHF to €128.20
    ```

5.  **Executing a forex trade as a speculative position**:
    ```bash demo
    pcs init -d 2025-01-01 -c USD
    pcs deposit -d 2025-01-01 -a 50000 -c USD
    pcs convert -d 2025-10-10 -fc USD -fa 50000 -tc JPY -ta 7250000 -m "Speculative long JPY"
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Deposit $50,000.00
      • 2025-10-10: Convert $50,000.00 to ¥7,250,000
    ```

#### `declare`

Establishes the canonical mapping between a user-defined ticker and a unique Security ID, defining its currency and type.

* **Flags**:
    * `-d`: (Optional) Transaction date. Defaults to the current day.
    * `-s`: (Required) Ledger-internal ticker to define.
    * `-id`: (Required) Full, unique security ID.
    * `-c`: (Required) The currency of the security.
    * `-m`: (Optional) A descriptive memo for the transaction.

1.  **Declaring a US-listed stock**:
    ```bash demo
    pcs init -d 2025-01-01 -c USD
    pcs declare -d 2025-01-01 -s AAPL -id US0378331005.XNAS -c USD
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Declare "AAPL" as "US0378331005.XNAS" in USD
    ```

2.  **Declaring a European ETF traded on XETRA**:
    ```bash demo
    pcs init -d 2025-01-01 -c EUR
    pcs declare -d 2025-01-01 -s IWDA -id IE00B4L5Y983.XETR -c EUR
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Declare "IWDA" as "IE00B4L5Y983.XETR" in EUR
    ```

3.  **Declaring a private equity fund**:
    ```bash demo
    pcs init -d 2025-01-01 -c USD
    pcs declare -d 2025-01-01 -s MyPEFund -id "PE-Fund-Vintage-2025" -c USD -m "Private Equity Fund X"
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Declare "MyPEFund" as "PE-Fund-Vintage-2025" in USD
    ```

4.  **Declaring the EUR/USD forex pair**:
    ```bash demo
    pcs init -d 2025-01-01 -c USD
    pcs declare -d 2025-01-01 -s EURUSD -id EURUSD -c USD
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Declare "EURUSD" as "EURUSD" in USD
    ```

5.  **Declaring a real estate asset using a private ID**:
    ```bash demo
    pcs init -d 2025-01-01 -c EUR
    pcs declare -d 2025-01-01 -s PrimaryResidence -id "RealEstate-MainSt-123" -c EUR
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Declare "PrimaryResidence" as "RealEstate-MainSt-123" in EUR
    ```

#### `deposit`

Records an external capital injection into a cash account, optionally settling a counterparty receivable.

* **Flags**:
    * `-d`: (Optional) Transaction date. Defaults to the current day.
    * `-a`: (Required) Amount of cash to deposit.
    * `-c`: (Optional) Currency of the deposit. Defaults to "EUR".
    * `-m`: (Optional) A descriptive memo for the transaction.
    * `-settles`: (Optional) The counterparty account this deposit settles.

1.  **Capital contribution to the investment portfolio**:
    ```bash demo
    pcs init -d 2025-01-01 -c EUR
    pcs deposit -d 2025-01-15 -a 50000 -c EUR -m "Annual portfolio funding"
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      • 2025-01-15: Deposit €50,000.00
    ```

2.  **Receiving a loan repayment from a counterparty**:
    ```bash demo
    pcs init -d 2025-01-01 -c USD
    pcs accrue -d 2025-01-01 -receivable PrivateLoan_JohnDoe -a 1000 -c USD
    pcs deposit -d 2025-05-15 -a 500 -c USD -settles PrivateLoan_JohnDoe -m "Loan principal repayment"
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    A new counterparty account 'PrivateLoan_JohnDoe' has been created.
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Accrue receivable $1,000.00 from "PrivateLoan_JohnDoe"
      • 2025-05-15: Deposit $500.00
    ```

3.  **Depositing proceeds from an external asset sale (e.g., a car)**:
    ```bash demo
    pcs init -d 2025-01-01 -c EUR
    pcs deposit -d 2025-09-20 -a 15000 -c EUR -m "Proceeds from car sale"
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      • 2025-09-20: Deposit €15,000.00
    ```

4.  **Receiving payment against a freelance invoice**:
    ```bash demo
    pcs init -d 2025-01-01 -c USD
    pcs accrue -d 2025-05-20 -receivable ClientX -a 2500 -c USD -m "Invoice #INV-2025-05-20"
    pcs deposit -d 2025-06-01 -a 2500 -c USD -settles ClientX -m "Payment for INV-2025-05-20"
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    A new counterparty account 'ClientX' has been created.
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      • 2025-05-20: Accrue receivable $2,500.00 from "ClientX"
      • 2025-06-01: Deposit $2,500.00
    ```

5.  **Settlement of an accrued bonus**:
    ```bash demo
    pcs init -d 2025-01-01 -c EUR
    pcs accrue -d 2025-12-31 -receivable EmployerCorp -a 10000 -c EUR -m "FY2025 performance bonus"
    pcs deposit -d 2026-01-31 -a 10000 -c EUR -settles EmployerCorp -m "Bonus payout"
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    A new counterparty account 'EmployerCorp' has been created.
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      • 2025-12-31: Accrue receivable €10,000.00 from "EmployerCorp"
      • 2026-01-31: Deposit €10,000.00
    ```

#### `dividend`

Records dividend income per share for a security, which contributes to total return without directly affecting the portfolio's cash balance.

* **Flags**:
    * `-d`: (Optional) Transaction date. Defaults to the current day.
    * `-s`: (Required) Security ticker receiving the dividend.
    * `-a`: (Required) Total dividend amount received.
    * `-m`: (Optional) A descriptive memo for the transaction.

1.  **Quarterly cash dividend**:
    ```bash demo
    pcs init -d 2025-01-01 -c USD
    pcs declare -d 2025-01-01 -s JNJ -id US4781601046.XNYS -c USD
    pcs deposit -d 2025-01-01 -a 10000 -c USD
    pcs buy -d 2025-01-10 -s JNJ -q 50 -a 8000
    pcs dividend -d 2025-03-15 -s JNJ -a 1.19
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Declare "JNJ" as "US4781601046.XNYS" in USD
      •           : Deposit $10,000.00
      • 2025-01-10: Buy 50 of "JNJ" for $8,000.00
      • 2025-03-15: Receive dividend of $1.19 per share for "JNJ"
    ```

2.  **Annual ETF distribution**:
    ```bash demo
    pcs init -d 2025-01-01 -c USD
    pcs declare -d 2025-01-01 -s VTSAX -id US9229087286.XNAS -c USD
    pcs deposit -d 2025-01-01 -a 20000 -c USD
    pcs buy -d 2025-03-01 -s VTSAX -q 100 -a 11000
    pcs dividend -d 2025-12-20 -s VTSAX -a 2.50
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Declare "VTSAX" as "US9229087286.XNAS" in USD
      •           : Deposit $20,000.00
      • 2025-03-01: Buy 100 of "VTSAX" for $11,000.00
      • 2025-12-20: Receive dividend of $2.50 per share for "VTSAX"
    ```

3.  **Special one-time dividend**:
    ```bash demo
    pcs init -d 2025-01-01 -c USD
    pcs declare -d 2025-01-01 -s MSFT -id US5949181045.XNAS -c USD
    pcs deposit -d 2025-01-01 -a 50000 -c USD
    pcs buy -d 2025-02-01 -s MSFT -q 100 -a 40000
    pcs dividend -d 2025-06-05 -s MSFT -a 3.00 -m "Special dividend from cash repatriation"
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Declare "MSFT" as "US5949181045.XNAS" in USD
      •           : Deposit $50,000.00
      • 2025-02-01: Buy 100 of "MSFT" for $40,000.00
      • 2025-06-05: Receive dividend of $3.00 per share for "MSFT"
    ```

4.  **Dividend from a foreign stock (in its local currency)**:
    ```bash demo
    pcs init -d 2025-01-01 -c CAD
    pcs declare -d 2025-01-01 -s BNS.TO -id CA0641491075.XTSE -c CAD
    pcs deposit -d 2025-01-01 -a 10000 -c CAD
    pcs buy -d 2025-04-01 -s BNS.TO -q 100 -a 6500
    pcs dividend -d 2025-09-25 -s BNS.TO -a 1.03
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Declare "BNS.TO" as "CA0641491075.XTSE" in CAD
      •           : Deposit $10,000.00
      • 2025-04-01: Buy 100 of "BNS.TO" for $6,500.00
      • 2025-09-25: Receive dividend of $1.03 per share for "BNS.TO"
    ```

5.  **Simulating a dividend reinvestment plan (DRIP)**:
    ```bash demo
    pcs init -d 2025-01-01 -c USD
    pcs declare -d 2025-01-01 -s KO -id US1912161007.XNYS -c USD
    pcs deposit -d 2025-01-01 -a 10000 -c USD
    pcs buy -d 2025-01-15 -s KO -q 198.369 -a 10000
    pcs dividend -d 2025-04-10 -s KO -a 0.46
    pcs deposit -d 2025-04-10 -a 91.25 -c USD -m "DRIP"
    pcs buy -d 2025-04-10 -s KO -q 1.5625 -a 91.25 -m "DRIP"
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Declare "KO" as "US1912161007.XNYS" in USD
      •           : Deposit $10,000.00
      • 2025-01-15: Buy 198.369 of "KO" for $10,000.00
      • 2025-04-10: Receive dividend of $0.46 per share for "KO"
      •           : Deposit $91.25
      •           : Buy 1.5625 of "KO" for $91.25
    ```

#### `init`

Establishes the ledger's fundamental parameters, including its inception date and reporting currency.

* **Flags**:
    * `-d`: (Optional) Inception date of the ledger.
    * `-c`: (Required) The reporting currency for the entire ledger.
    * `-m`: (Optional) A descriptive memo for the transaction.

1.  **Starting a new portfolio from scratch**:
    ```bash demo
    pcs init -d 2020-01-01 -c EUR
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2020-01-01: init
    ```

2.  **Initializing with a different base currency**:
    ```bash demo
    pcs init -d 2018-05-15 -c USD -m "Portfolio inception"
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2018-05-15: init
    ```

3.  **Setting up a ledger for a new tax year**:
    ```bash demo
    pcs init -d 2024-04-06 -c GBP
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2024-04-06: init
    ```

4.  **Creating a separate portfolio for a different entity**:
    ```bash demo
    pcs init -d 2021-01-01 -c CHF -m "Child's trust fund"
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2021-01-01: init
    ```

5.  **Correcting the inception date of an existing ledger**:
    ```bash demo
    pcs init -d 2025-01-01 -c EUR
    pcs init -d 2019-12-31 -c EUR
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2019-12-31: init
      • 2025-01-01: init
    ```

#### `price`

Logs a market price point for a security on a specific date, essential for mark-to-market valuation.

* **Flags**:
    * `-d`: (Optional) Date of the price. Defaults to the current day.
    * `-s`: (Required) Security ticker.
    * `-p`: (Required) Price per share.

1.  **Updating the daily closing price of a stock**:
    ```bash demo
    pcs init -d 2025-01-01 -c USD
    pcs declare -d 2025-01-01 -s GOOG -id US38259P5089.XNAS -c USD
    pcs price -d 2025-03-10 -p 140.25 -s GOOG
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Declare "GOOG" as "US38259P5089.XNAS" in USD
      • 2025-03-10: Update price for "GOOG"=140.2500
    ```

2.  **Recording the Net Asset Value (NAV) of a private fund**:
    ```bash demo
    pcs init -d 2025-01-01 -c USD
    pcs declare -d 2025-01-01 -s MyPEFund -id "PE-Fund-Vintage-2025" -c USD
    pcs price -d 2025-12-31 -p 1.35 -s MyPEFund
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Declare "MyPEFund" as "PE-Fund-Vintage-2025" in USD
      • 2025-12-31: Update price for "MyPEFund"=1.3500
    ```

3.  **Updating the valuation of a real estate asset**:
    ```bash demo
    pcs init -d 2025-01-01 -c EUR
    pcs declare -d 2025-01-01 -s PrimaryResidence -id "RealEstate-MainSt-123" -c EUR
    pcs price -d 2025-06-30 -p 750000 -s PrimaryResidence
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Declare "PrimaryResidence" as "RealEstate-MainSt-123" in EUR
      • 2025-06-30: Update price for "PrimaryResidence"=750000.0000
    ```

4.  **Recording a historical forex rate**:
    ```bash demo
    pcs init -d 2024-01-01 -c EUR
    pcs declare -d 2024-01-01 -s USDEUR -id USDEUR -c EUR
    pcs price -d 2024-01-01 -p 0.9050 -s USDEUR
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2024-01-01: init
      •           : Declare "USDEUR" as "USDEUR" in EUR
      •           : Update price for "USDEUR"=0.9050
    ```

5.  **Correcting an erroneous price from a data feed**:
    ```bash demo
    pcs init -d 2025-01-01 -c USD
    pcs declare -d 2025-01-01 -s F -id US3453708600.XNYS -c USD
    pcs price -d 2025-02-15 -p 12.05 -s F
    # Correct the price
    pcs price -d 2025-02-15 -p 12.50 -s F
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Declare "F" as "US3453708600.XNYS" in USD
      • 2025-02-15: Update price for "F"=12.0500
      •           : Update price for "F"=12.5000
    ```

#### `sell`

Records the disposition of a security, triggering a realized gain or loss calculation and crediting the corresponding cash account.

* **Flags**:
    * `-d`: (Optional) Transaction date. Defaults to the current day.
    * `-s`: (Required) Security ticker.
    * `-a`: (Required) Total amount received for the shares.
    * `-q`: (Optional) Number of shares to sell. If omitted, all shares of the security are sold.
    * `-m`: (Optional) A descriptive memo for the transaction.

1.  **Selling a portion of a holding to take profits**:
    ```bash demo
    pcs init -d 2025-01-01 -c USD
    pcs declare -d 2025-01-01 -s NVDA -id US67066G1040.XNAS -c USD
    pcs deposit -d 2025-01-01 -a 25000 -c USD
    pcs buy -d 2025-01-10 -s NVDA -q 50 -a 15000
    pcs sell -d 2025-05-20 -s NVDA -q 25 -a 20050.00
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Declare "NVDA" as "US67066G1040.XNAS" in USD
      •           : Deposit $25,000.00
      • 2025-01-10: Buy 50 of "NVDA" for $15,000.00
      • 2025-05-20: Sell 25 of "NVDA" for $20,050.00
    ```

2.  **Liquidating an entire position**:
    ```bash demo
    pcs init -d 2024-01-15 -c USD
    pcs declare -d 2024-01-15 -s BABA -id US01609W1027.XNYS -c USD
    pcs deposit -d 2024-01-15 -a 30000 -c USD
    pcs buy -d 2024-01-15 -s BABA -q 200 -a 28000
    pcs sell -d 2025-09-15 -s BABA -a 25300.00 -m "Exiting position"
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2024-01-15: init
      •           : Declare "BABA" as "US01609W1027.XNYS" in USD
      •           : Deposit $30,000.00
      •           : Buy 200 of "BABA" for $28,000.00
      • 2025-09-15: Sell 200 of "BABA" for $25,300.00
    ```

3.  **Selling an ETF for portfolio rebalancing**:
    ```bash demo
    pcs init -d 2025-01-01 -c USD
    pcs declare -d 2025-01-01 -s EEM -id US4642872349.XNAS -c USD
    pcs deposit -d 2025-01-01 -a 5000 -c USD
    pcs buy -d 2025-06-01 -s EEM -q 100 -a 3800
    pcs sell -d 2025-12-01 -s EEM -q 100 -a 4100.00
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Declare "EEM" as "US4642872349.XNAS" in USD
      •           : Deposit $5,000.00
      • 2025-06-01: Buy 100 of "EEM" for $3,800.00
      • 2025-12-01: Sell 100 of "EEM" for $4,100.00
    ```

4.  **Selling shares to cover taxes upon vesting of RSUs**:
    ```bash demo
    pcs init -d 2025-01-01 -c USD
    pcs declare -d 2025-01-01 -s ADBE -id US00724F1012.XNAS -c USD
    pcs deposit -d 2025-11-15 -a 10250.00 -c USD -m "RSU Vesting"
    pcs buy -d 2025-11-15 -s ADBE -q 400 -a 10250.00 -m "RSU Vesting"
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Declare "ADBE" as "US00724F1012.XNAS" in USD
      • 2025-11-15: Deposit $10,250.00
      •           : Buy 400 of "ADBE" for $10,250.00
    ```

5.  **Tax-loss harvesting at year-end**:
    ```bash demo
    pcs init -d 2025-01-01 -c USD
    pcs declare -d 2025-01-01 -s PYPL -id US70450Y1038.XNAS -c USD
    pcs deposit -d 2025-01-01 -a 10000 -c USD
    pcs buy -d 2025-01-20 -s PYPL -q 100 -a 9500
    pcs sell -d 2025-12-28 -s PYPL -a 8500.00 -m "Tax-loss harvest"
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Declare "PYPL" as "US70450Y1038.XNAS" in USD
      •           : Deposit $10,000.00
      • 2025-01-20: Buy 100 of "PYPL" for $9,500.00
      • 2025-12-28: Sell 100 of "PYPL" for $8,500.00
    ```

#### `split`

Adjusts the quantity of all existing lots for a security to reflect a corporate action, preserving the total cost basis.

* **Flags**:
    * `-d`: (Optional) Effective date of the split. Defaults to the current day.
    * `-s`: (Required) Security ticker.
    * `-num`: (Required) The numerator of the split ratio.
    * `-den`: (Optional) The denominator of the split ratio. Defaults to 1.

1.  **A 2-for-1 stock split**:
    ```bash demo
    pcs init -d 2025-01-01 -c USD
    pcs declare -d 2025-01-01 -s AAPL -id US0378331005.XNAS -c USD
    pcs split -d 2025-08-31 -s AAPL -num 2 -den 1
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Declare "AAPL" as "US0378331005.XNAS" in USD
      • 2025-08-31: split
    ```

2.  **A 1-for-8 reverse stock split**:
    ```bash demo
    pcs init -d 2025-01-01 -c USD
    pcs declare -d 2025-01-01 -s GE -id US3696041033.XNYS -c USD
    pcs split -d 2025-07-19 -s GE -num 1 -den 8
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Declare "GE" as "US3696041033.XNYS" in USD
      • 2025-07-19: split
    ```

3.  **A 3-for-2 stock split**:
    ```bash demo
    pcs init -d 2025-01-01 -c USD
    pcs declare -d 2025-01-01 -s NVDA -id US67066G1040.XNAS -c USD
    pcs split -d 2025-05-21 -s NVDA -num 3 -den 2
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Declare "NVDA" as "US67066G1040.XNAS" in USD
      • 2025-05-21: split
    ```

4.  **A 20-for-1 stock split**:
    ```bash demo
    pcs init -d 2025-01-01 -c USD
    pcs declare -d 2025-01-01 -s AMZN -id US0231351067.XNAS -c USD
    pcs split -d 2025-06-03 -s AMZN -num 20 -den 1
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Declare "AMZN" as "US0231351067.XNAS" in USD
      • 2025-06-03: split
    ```

5.  **A 1-for-10 reverse stock split**:
    ```bash demo
    pcs init -d 2025-01-01 -c USD
    pcs declare -d 2025-01-01 -s CITI -id US1729674242.XNYS -c USD
    pcs split -d 2025-05-09 -s CITI -num 1 -den 10
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Declare "CITI" as "US1729674242.XNYS" in USD
      • 2025-05-09: split
    ```

#### `withdraw`

Records an external capital withdrawal from a cash account, optionally settling a counterparty payable.

* **Flags**:
    * `-d`: (Optional) Transaction date. Defaults to the current day.
    * `-a`: (Required) Amount of cash to withdraw.
    * `-c`: (Optional) Currency of the withdrawal. Defaults to "EUR".
    * `-m`: (Optional) A descriptive memo for the transaction.
    * `-settles`: (Optional) The counterparty account this withdrawal settles.

1.  **Withdrawing funds for personal expenses**:
    ```bash demo
    pcs init -d 2025-01-01 -c EUR
    pcs deposit -d 2025-01-01 -a 10000 -c EUR
    pcs withdraw -d 2025-03-05 -a 5000 -c EUR -m "Living expenses"
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Deposit €10,000.00
      • 2025-03-05: Withdraw €5,000.00
    ```

2.  **Making a loan payment to a counterparty**:
    ```bash demo
    pcs init -d 2025-01-01 -c USD
    pcs deposit -d 2025-01-01 -a 2000 -c USD
    pcs accrue -d 2025-01-15 -payable PrivateLoan_JohnDoe -a 1000 -c USD
    pcs withdraw -d 2025-04-15 -a 500 -c USD -settles PrivateLoan_JohnDoe -m "Loan principal payment"
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    A new counterparty account 'PrivateLoan_JohnDoe' has been created.
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Deposit $2,000.00
      • 2025-01-15: Accrue payable $1,000.00 to "PrivateLoan_JohnDoe"
      • 2025-04-15: Withdraw $500.00
    ```

3.  **Withdrawing for a large purchase (e.g., car)**:
    ```bash demo
    pcs init -d 2025-01-01 -c EUR
    pcs deposit -d 2025-01-01 -a 50000 -c EUR
    pcs withdraw -d 2025-08-01 -a 40000 -c EUR -m "Car purchase"
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Deposit €50,000.00
      • 2025-08-01: Withdraw €40,000.00
    ```

4.  **Paying an accrued tax liability**:
    ```bash demo
    pcs init -d 2025-01-01 -c EUR
    pcs deposit -d 2025-01-01 -a 20000 -c EUR
    pcs accrue -d 2025-12-31 -payable TaxAuthority -a 15200 -c EUR
    pcs withdraw -d 2026-04-15 -a 15200 -c EUR -settles TaxAuthority -m "Payment for FY2025 taxes"
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    A new counterparty account 'TaxAuthority' has been created.
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Deposit €20,000.00
      • 2025-12-31: Accrue payable €15,200.00 to "TaxAuthority"
      • 2026-04-15: Withdraw €15,200.00
    ```

5.  **Withdrawing foreign currency for a trip**:
    ```bash demo
    pcs init -d 2025-01-01 -c EUR
    pcs deposit -d 2025-01-01 -a 2000 -c EUR
    pcs convert -d 2025-07-01 -fc EUR -fa 1500 -tc JPY -ta 220000
    pcs withdraw -d 2025-07-10 -a 200000 -c JPY
    pcs tx
    ```
    ```console check
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    ✅ Successfully recorded transaction in ledger "transactions".
    
    
      • 2025-01-01: init
      •           : Deposit €2,000.00
      • 2025-07-01: Convert €1,500.00 to ¥220,000
      • 2025-07-10: Withdraw ¥200,000
    ```