# pcs Architecture and Guiding Principles

This document outlines the architecture of the `pcs` portfolio tool. It serves as the single source of truth for the project's design, conventions, and the relationship between its components. It is guided by the project's core value proposition, as defined in `Vision.md`. The primary purpose of this document is to ensure that all development, whether by human or AI contributors, adheres to a consistent and coherent vision, promoting long-term maintainability and clarity.

---
## 1. Architectural Goals

The design of `pcs` is guided by a set of core principles that inform all development decisions.

* **Local-First:** All user data is stored on the local filesystem. The tool must be fully functional without requiring access to cloud services for its core operations.
* **Auditable & Version-Controllable:** Data is stored in human-readable, line-oriented text files (`.jsonl`) to work seamlessly with version control systems like Git. This ensures that every change to a portfolio is transparent and auditable.
* **Extensible:** The system is designed to be extended with custom, external commands (e.g., `pcs-my-importer`) to support new data sources and integrations without modifying the core application.
* **Single Source of Truth:** The Go source code is the canonical source for all user-facing documentation. The user manual and other artifacts are derived from the code itself to prevent documentation drift.

---
## 2. Core Ontology (The Language)

This section defines the key conceptual entities that form the domain language of the portfolio. These are the "what" of the system—the concepts a user interacts with.

* **Ledger (`ledger.go`, `transactions.go`):** The immutable, chronological record of all user-initiated actions (buys, sells, deposits) and relevant market data (prices, splits, dividends). It represents the user's input and financial history, and contains all information required to assess the portfolio's total market value. This makes the ledger the single, self-contained source of truth.
* **Security ID (`type_id.go`):** The crucial, unambiguous link that decouples the user's personal, short-hand tickers from the global, canonical identifiers of financial assets.
* **Counterparty Account:** A core concept representing the financial balance with a specific external entity (e.g., "Landlord", "John Doe", "ClientX").
* **Portfolio Metrics:** These are the high-level, calculated insights derived from the Ledger. They answer the user's core questions about their wealth. Examples include:
    * **Holdings:** What assets are owned at a specific point in time.
    * **Total Portfolio Value:** The net worth of all assets.
    * **Cash Flow:** The movement of capital into and out of the portfolio.
    * **Gains & Losses:** Both realized (from sales) and unrealized (paper) gains.
    * **Time-Weighted Return (TWR):** The pure performance of the investment strategy, independent of cash flows.

---
## 3. Components (The Code Modules)

This section describes the main Go packages and their distinct responsibilities. These are the "how" of the system—the implementation details.

* **`portfolio` (Core Logic):** This is the main library package containing the implementation of the Core Ontology (`ledger.go`, `transactions.go`, `type_id.go`) and the business logic of the Calculation Engine. This package is completely decoupled from the user interface and data persistence layers.
* **`cmd` (User Interface):** This package implements the command-line interface. Each command is a thin wrapper that is responsible for parsing flags, validating user input, and calling the appropriate logic in the `portfolio` package to perform actions or calculations. It also contains the logic for discovering and executing external extension commands.
* **Calculation Engine (`journal.go`, `snapshot.go`, `review.go`):** The stateless "brains" of the application. It transforms the raw ledger into the **Portfolio Metrics** through a one-way data pipeline.
    * **Journal:** The `Ledger` is first processed into a `Journal`, a chronologically sorted list of atomic financial events. This serves as a canonical, intermediate representation.
    * **Snapshot:** A `Snapshot` is a stateless calculator that processes the `Journal` up to a specific point in time to determine the state of the portfolio (e.g., positions, cash balances, market values) on that day.
    * **Review:** A `Review` compares two `Snapshots` (at the beginning and end of a period) to calculate performance metrics over that range (e.g., Time-Weighted Return, market gains, cash flow).
* **Core Business Types (within `portfolio` package):** These are the value-object types that represent fundamental business concepts. They are located in `types_*.go` files (e.g., `type_id.go`, `types_money.go`). This includes `ID`, `Money`, `Quantity`, `Date`, `Range`, and `Period`.
* **Persistence (within `portfolio` package):** This component is responsible for all I/O operations with the `transactions.jsonl` file. It ensures data is read and written in a canonical, backward-compatible way. It is located in `encode_ledger.go`.
    * **Encoding (Writing):** To guarantee a stable, canonical output, each transaction type implements its own `MarshalJSON` method. These methods use a custom `jsonObjectWriter` to control the exact order and format of the JSON fields.
    * **Decoding (Reading):** The `DecodeLedger` function (in `encode_ledger.go`) reads the `.jsonl` file line-by-line. It uses a two-pass decoding strategy: first, it identifies the transaction `command`, then it unmarshals the line into the corresponding concrete transaction struct.
* **Market Data Providers:** These components are responsible for fetching market data (prices, splits, dividends) from third-party sources. They are the only parts of the application permitted to make network requests and are invoked by the `fetch` command. There are two types:
    * **Internal Providers (e.g., `eodhd`, `amundi`):** These are Go packages compiled directly into the `pcs` binary, offering built-in support for common data sources.
    * **External Providers (Future):** Following the general extension mechanism, `pcs` will support external providers as standalone executables (e.g., `pcs-fetch-myprovider`). This will allow the community to add support for any data source without modifying the core application.

* **`renderer` (Output Formatting):** This package contains helpers for generating user-facing output, primarily in Markdown format. It is used by the `cmd` package to display reports. A key utility is `renderer.ConditionalBlock`, which allows for the conditional printing of sections (e.g., printing a "Cash Accounts" table only if there are cash accounts to show), simplifying the logic for creating clean and readable reports.

---
## 4. Data View (The Files)

This section describes the physical layout and format of the data stored on disk.

* **`transactions.jsonl`:** Stores the **Ledger**. It is an append-only file where each line is a single JSON object representing a transaction (including personal transactions like buys/sells and market data like prices/splits). For canonical representation, the file is sorted by date.

---
## 5. Documentation and Artifacts
 
This section outlines the key documents that guide and describe the project. These are divided into two categories: developer-focused strategic documents and user-focused, test-verified guides.
 
### 5.1. Developer Documentation
 
These documents define the "why" and "how" of the project's construction and are intended for contributors.
 
* **`ARCHITECTURE.md` (This Document):** The technical blueprint. It describes the system's components, their responsibilities, and the principles that govern their interaction. It is the single source of truth for the project's design.
* **`Vision.md`:** The product strategy document. It defines the core value proposition, target audience, and long-term goals, guiding all feature development.
 
### 5.2. User Documentation (Test-Verified)
 
These documents are for the end-users of `pcs`. A key principle is that all user-facing documentation containing command examples is **testable and automatically verified**. The integration tests in `docs/topics_test.go` execute the code blocks within these markdown files to ensure they are always accurate and up-to-date with the current implementation. When a feature is updated, the corresponding documentation and tests must be updated in lockstep.
 
* **`README.md`:** The primary entry point for new users. It provides a high-level overview and a "Getting Started" tutorial.
* **User Manual Topics (`docs/*.md`):** A collection of detailed, topic-specific guides that form the complete user manual. They are accessible via the `pcs topic` command. The `docs/readme.md` file serves as the index for these topics.

#### The `-fix-docs` Workflow

To streamline the maintenance of these test-verified documents, the test suite includes a powerful utility flag: `-fix-docs`.

When a command's output changes, the `console check` blocks in the documentation will become outdated, causing tests to fail. Instead of manually updating each markdown file, a developer can run `go test ./docs -fix-docs`. This command will execute the tests, and for any failing `console check` block, it will automatically replace the old content with the new, correct output directly in the source `.md` file.

This mechanism is a cornerstone of the "documentation as code" philosophy, ensuring that keeping the user manual synchronized with the application's behavior is a low-friction, automated process.
---
## 6. Development Workflow

This section outlines the collaborative process for extending the tool, ensuring that architectural integrity is maintained. The process is divided into two distinct phases.

### 6.1. The Design Phase

This phase is for analyzing a feature request, understanding its architectural impact, and getting explicit approval for any necessary architectural changes *before* any code is written.

1.  **The Request:** The architect provides a high-level feature request.
2.  **AI Analysis & Proposal:** The assistant's first task is to create a **Design Proposal** by analyzing the request against this `ARCHITECTURE.md` document.
3.  **The Design Proposal Document:** The proposal must include an **Architectural Impact Assessment** and, if necessary, **Proposed Architectural Changes** to this document. The implementation plan should be broken into distinct steps, separating architectural changes from feature implementation.
4.  **Architect's Approval:** The implementation phase does not begin until the architect explicitly approves the design.

### 6.2. The Implementation Phase

This phase is for executing the approved design plan with precision.

1.  **Execution:** The implementation plan serves as a guide, but the developer has the freedom to adapt to technical realities discovered during coding.
2.  **Adherence to Architecture:** During this phase, the developer's primary directive is to **strictly adhere to the approved architecture**. No unapproved architectural changes are to be made on the fly.
3.  **Responsibility:** The developer is responsible for updating all relevant artifacts. The **Components** and **Documentation and Artifacts** sections of this document serve as a checklist for what may need to be updated.

---
## 7. Development Checklists

### 7.1. Checklist for Adding a New Report

This checklist provides a step-by-step guide for adding a new report command to `pcs`. Following these steps ensures that all parts of the architecture, from core logic to user-facing documentation, are updated consistently.

#### Phase 1: Core Logic and Rendering

1.  **Determine Data Requirements**:
    * [ ] Analyze what information the new report needs.
    * [ ] Verify if the required data can be derived from the existing `portfolio.Snapshot` and `portfolio.Review` types. Most reports should be able to use these as their foundation.

2.  **Implement Core Calculations (if necessary)**:
    * [ ] If new calculations are needed, add them as methods to `portfolio.Snapshot` or `portfolio.Review` in the `portfolio` package. This keeps the core business logic centralized.

3.  **Create the Markdown Renderer**:
    * [ ] Create a new file in the `renderer` package (e.g., `renderer/new_report.go`).
    * [ ] Implement a new function (e.g., `NewReportMarkdown`) that takes the necessary data structure (like `*portfolio.Review`) and returns a formatted markdown `string`.

#### Phase 2: Command-Line Interface

4.  **Create the Command File**:
    * [ ] Create a new file for the command in the `cmd` package (e.g., `cmd/new_report.go`).

5.  **Implement the Command Struct**:
    * [ ] Define a `newReportCmd` struct.
    * [ ] Implement the `subcommands.Command` interface for the struct (`Name`, `Synopsis`, `Usage`, `SetFlags`, `Execute`).

6.  **Implement the `Execute` Method**:
    * [ ] Write the logic within `Execute` to:
        * Parse and validate command-line flags.
        * Load the ledger using `cmd.DecodeLedger()`.
        * Call the appropriate core logic function from the `portfolio` package to perform the calculations.
        * Pass the result to the new renderer function to get the markdown output.
        * Print the final output using `cmd.printMarkdown()`.

7.  **Register the New Command**:
    * [ ] Open `cmd/app.go`.
    * [ ] Add the new command to the `c.Register(...)` list, placing it in the "reports" group.

#### Phase 3: Documentation and Integration Testing

8.  **Create the Documentation Topic**:
    * [ ] Create a new markdown file in the `docs` directory (e.g., `docs/new-report.md`). The filename must match the topic name you will use.

9.  **Write Comprehensive User Documentation**:
    * [ ] In the new file, clearly explain the purpose of the report, the key metrics it displays, and how they are calculated.

10. **Add Testable Scenarios**:
    * [ ] Include at least one complete, working scenario using the `bash setup`, `bash run`, and `console check` code blocks. This is a critical step as these blocks serve as the integration tests for the new command.

11. **Update the Documentation Index**:
    * [ ] Open `docs/readme.md`.
    * [ ] Add the new topic (the filename without `.md`) and a brief description to the "Available Topics" list.

12. **Verify and Validate**:
    * [ ] Run the integration tests for the documentation by executing `go test ./docs`.
    * [ ] Ensure all tests, including the new scenario, pass successfully. If they fail, update the `console check` block with the correct output.
