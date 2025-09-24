# Use Cases for the `pcs` Portfolio Tracking Tool

This document outlines the key problems that long-term investors face and how `pcs` aims to solve them. It serves as both a guide to the tool's current capabilities and a roadmap for its future development.

The use cases are framed by the core value propositions of `pcs`: **Unification**, **Durability**, and **Clarity**. Some of these are fully realized today, while others represent future goals. The goal is to grow `pcs` to become more and more helpful in each of these areas over time.

### Coverage Rating

Each use case includes a rating to provide a clear picture of its current maturity within `pcs`.

* **Core:** How well the fundamental `pcs` commands and data structures support the use case.

    * ⭐️⭐️⭐️: Fully supported by core features.
    * ⭐️⭐️★: Largely supported, but may require some user scripting or workarounds.
    * ⭐️★★: Partially supported; a key goal for future development.
    * ★★★: Not yet supported; represents a future goal.

* **Ecosystem:** The maturity of the extension ecosystem (data providers, AI integrations) for this use case. This is independent of the core tool's capabilities.

    * **High / Medium / Low:** Reflects the availability and breadth of ready-to-use extensions.

## 1. Use Cases for Unification (A Single View of Your Wealth)

`pcs` is designed to consolidate every scattered piece of your wealth, giving you a single, coherent picture of your net worth.

### The Scattered Portfolio

**Problem:** My wealth is scattered across multiple brokers and includes a mix of public stocks, ETFs, and private funds from my company's savings plan. Getting a single view of my total investments is a manual, error-prone process.

**`pcs` Solution:** `pcs` unifies these disparate assets into a single ledger. You can track everything in one place and run consolidated reports.

**Coverage:**

* **Core:** ⭐️⭐️⭐️ (The ledger is designed specifically for this.)
* **Ecosystem:** Medium (`eodhd` covers most public securities, `amundi` provides a template for private funds, but importing from every broker requires custom extensions.)

### The Alternative Asset Blind Spot

**Problem:** A significant portion of my net worth is in alternative assets like real estate or collectibles. These assets are tracked differently from financial securities; they don't have a daily market price, and their value must be estimated between infrequent, official valuations (like a purchase or an expert appraisal).

**`pcs` Solution:** `pcs` can track any asset. The recommended method for alternative assets is to use a market index as a proxy for its value. For example, you can record the purchase of a house and then track its value against a regional housing index provided by an extension (like `insee`). When a new, more accurate valuation is available (e.g., from a professional appraiser), you can re-calibrate your holding by recording a `split` transaction. This adjusts the number of "shares" of the index you hold to match the new valuation, ensuring historical accuracy is maintained.

**Coverage:**

* **Core:** ⭐️⭐️★ (The index-proxy model is fully supported, but using `split` to re-calibrate valuations is not intuitive. A dedicated `revalue` command would be a future improvement.)
* **Ecosystem:** Low (The built-in `insee` provider for French real estate serves as a powerful proof-of-concept, but tracking other alternative assets requires new, specific extensions.)

### The Currency Maze

**Problem:** I hold stocks in USD, funds in EUR, and cash in JPY. It's difficult to get a true measure of my global net worth without manually converting everything.

**`pcs` Solution:** `pcs` seamlessly tracks assets and cash balances in any number of currencies. It automatically handles the conversion to your chosen reporting currency for all reports, providing an accurate, real-time view of your total wealth.

**Coverage:**

* **Core:** ⭐️⭐️⭐️ (Fully supported.)
* **Ecosystem:** Medium (While the `eodhd` provider offers excellent forex coverage, the ecosystem rating reflects the dependency on a single source. A higher rating would require multiple competing providers to ensure long-term durability and avoid vendor lock-in.)

### The Family Financial Puzzle

**Problem:** I manage my personal finances, a joint account with my spouse, and a small portfolio for my children. I need to analyze these "financial compartments" separately but also see a consolidated family view.

**`pcs` Solution:** The local-first design allows you to manage separate ledger files for each compartment (`personal.jsonl`, `joint.jsonl`). You can run reports on each file individually or use simple scripts to concatenate them for a combined family wealth overview. However, transfers between these ledgers (e.g., sending money from a personal account to a child's account) would currently be recorded as a `withdraw` and a `deposit`, incorrectly inflating the global cash flow. A future enhancement would be a dedicated `transfer` transaction to correctly handle these internal movements.

**Coverage:**

* **Core:** ⭐️⭐️★ (Supported via multiple files, but requires user scripting to merge and lacks a mechanism to handle inter-ledger transfers correctly, which distorts global cash flow. A built-in aggregation feature and `transfer` command would improve this.)
* **Ecosystem:** Not Applicable.

### The Hidden Liabilities

**Problem:** My net worth isn't just my assets; it's also my liabilities like a mortgage or a tax bill I need to pay. Ignoring these gives me an inflated and inaccurate sense of my true financial position.

**`pcs` Solution:** `pcs` allows you to track liabilities through "counterparty accounts." You can manually record a mortgage, a personal loan, or an accrued tax bill and update it with each payment. For full automation, bank-specific extensions could be developed to fetch loan statements and update the balance automatically.

**Coverage:**

* **Core:** ⭐️⭐️⭐️ (Counterparty accounts are a built-in feature for manual tracking.)
* **Ecosystem:** Low (Automating loan balance updates is dependent on building bank-specific extensions, much like fetching private fund data.)

### The Employee Stock Option Dilemma

**Problem:** A large part of my future wealth is tied up in employee stock options, but they aren't part of my net worth until they vest. I want to track this "potential wealth" without distorting my current financial picture.

**`pcs` Solution:** You can maintain two separate ledgers: one for your actual net worth (`net_worth.jsonl`) and another for unvested grants (`potential_wealth.jsonl`). This provides a clear separation. For automation, a provider-specific extension could be developed to parse vesting reports, automatically moving shares from the 'potential' ledger to the 'net worth' ledger and accounting for shares sold for taxes.

**Coverage:**

* **Core:** ⭐️★★ (Possible via the two-ledger workaround, but `pcs` lacks dedicated commands or types for options and vesting schedules.)
* **Ecosystem:** Low (Automating the transfer of vested shares is dependent on building institution-specific extensions to parse grant and vesting reports.)

### Tracking Annual Income

**Problem:** I want to track my annual income to calculate my savings rate and get a clearer picture of my financial health, but this data often lives in proprietary HR portals.

**`pcs` Solution:** `pcs` provides multiple pathways to incorporate this crucial data point. Users can choose the method that best fits their needs:

1.  **Manual Entry:** A simple, once-a-year manual transaction can record the total income.
2.  **Dedicated Extensions:** For companies with accessible data, a specific extension could be built to automate this process.
3.  **AI as a Universal Adapter:** This is the most scalable solution. Instead of building hundreds of extensions, a user can feed a document like an annual pay letter to a well-prompted AI. The AI, understanding `pcs`'s format, can parse the document and generate the correct transactions, effectively serving as a universal importer for any source.

**Coverage:**

* **Core:** ★★★ (Tracking income is not a core feature. This is a future goal that would require dedicated transaction types to be properly supported.)
* **Ecosystem:** Low (Requires users to build their own LLM pipelines.)

## 2. Use Cases for Durability (A Permanent, Trustworthy Record)

### The Fear of Platform Lock-In

**Problem:** Cloud-based portfolio trackers can shut down, change their terms, or get acquired, forcing me into a painful data migration. I'm worried about losing decades of financial history.

**`pcs` Solution:** `pcs` is local-first. Your `transactions.jsonl` file lives on your machine, in a simple, human-readable format. By storing it in a Git repository, you create a permanent, auditable, and future-proof record of your wealth that you control completely. For interoperability, extensions can be built to import from (for easy onboarding) or export to (for backup or use with other tools) existing cloud-based trackers, ensuring you are never locked in.

**Coverage:**

* **Core:** ⭐️⭐️⭐️ (A fundamental design principle.)
* **Ecosystem:** Low (The concept of import/export extensions is a key part of the vision, but no such extensions exist today.)

### The Shifting Sands of Data Providers

**Problem:** The market data provider I use today might not be the best or most cost-effective one in five years. I don't want my financial history to be locked into a single provider's ecosystem.

**`pcs` Solution:** `pcs` is provider-independent. The tool's core logic is completely decoupled from the data sources. You can start with `eodhd`, switch to a competitor by writing a new extension, or fall back to manual updates with `pcs price` at any time. Your data remains yours.

**Coverage:**

* **Core:** ⭐️⭐️⭐️ (A fundamental design principle.)
* **Ecosystem:** Low (The small number of current providers highlights the importance of this independence, as the ecosystem needs to grow.)

### The Tyranny of Manual Entry

**Problem:** Manually entering transactions from brokerage statements, especially years of history, is the single biggest barrier to maintaining an accurate portfolio.

**`pcs` Solution:** The simple, text-based nature of `pcs` makes it a perfect partner for AI. A user can feed screenshots or PDF reports from their broker into a multimodal LLM (like Gemini) and instruct it to generate the `pcs` commands needed to import transaction history. Future `pcs` features or extensions could even help generate the optimal prompts for the AI to ensure accurate parsing, turning hours of tedious work into minutes.

**Coverage:**

* **Core:** ⭐️⭐️⭐️ (The CLI and simple data format are perfectly suited for this.)
* **Ecosystem:** Low (Requires users to build their own LLM pipelines; no integrated prompt-generation tools or extensions exist yet.)

## 3. Use Cases for Clarity (Answering the Questions That Matter)

### Good Strategy or Just Good Timing?

**Problem:** I know my portfolio value went up, but I don't know if that's because I'm a good investor or just because I deposited a lot of cash. I need to distinguish my strategy's performance from my cash flows.

**`pcs` Solution:** `pcs` uses the Time-Weighted Return (TWR) method, the industry standard for performance measurement. It calculates the return of your underlying strategy, removing the distorting effects of cash inflows and outflows, so you can accurately assess your investment choices.

**Coverage:**

* **Core:** ⭐️⭐️⭐️ (A built-in reporting feature.)
* **Ecosystem:** Not Applicable.

### The Cost of Living Mystery

**Problem:** I don't want to track every single coffee I buy, but I still want to know my approximate cost of living and whether I'm saving enough or starting to live off my investments.

**`pcs` Solution:** `pcs` provides high-level clarity without tedious bookkeeping. By comparing your known annual income (tracked externally) against the yearly `Cash Flow` calculated by a `pcs review` report, you can easily derive your effective cost of living and see if you're a net saver or beginning to draw down your portfolio. Adding a dedicated way to track income streams is a goal for future development.

**Coverage:**

* **Core:** ⭐️⭐️★ (The necessary portfolio-side data (Cash Flow) is available in reports, but tracking income itself is not yet a feature, making the final comparison a manual step.)
* **Ecosystem:** Low (Automating income tracking would require custom extensions for specific company payroll systems.)

### The "Story Behind the Numbers"

**Problem:** My portfolio report gives me the numbers, but they lack context. I want a qualitative summary of what happened, enriched with market news and presented in a more engaging way than a terminal table.

**`pcs` Solution:** The output of any `pcs` command can be piped directly into an LLM. You can feed a `pcs log` report into an AI and ask it to "summarize the highlights and lowlights of this period, correlating performance with major market news" or "generate a web dashboard visualizing these results." `pcs` provides the clean, structured data; AI provides the narrative and presentation layer.

**Coverage:**

* **Core:** ⭐️⭐️⭐️ (The CLI and simple data format are perfectly suited for this.)
* **Ecosystem:** Low (Requires users to build their own LLM pipelines and prompts.)

### The Fog of the Future

**Problem:** I need to plan for the future, but simple simulations feel like guesswork. I want to see how my portfolio would be affected by real-world events like retirement, an inheritance, or my stock options vesting.

**`pcs` Solution:** This is a key area for future development. The vision is for `pcs` to use a dedicated scenario file to model future events. It will generate simulated ledgers based on these scenarios, allowing you to run standard reports on your projected future wealth and make more informed long-term decisions.

**Coverage:**

* **Core:** ★★★ (This is a future goal. The current workaround of using manual scripting and separate files is considered outside the scope of core support.)
* **Ecosystem:** Not Applicable.