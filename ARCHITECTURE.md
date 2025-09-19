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

This section defines the key conceptual entities that form the domain language of the portfolio.

* **Ledger (`ledger.go`, `transactions.go`):** The immutable, chronological record of all user-initiated actions (buys, sells, deposits) and relevant market data (prices, splits, dividends). It represents the user's input and financial history, and contains all information required to assess the portfolio's total market value. This makes the ledger the single, self-contained source of truth.
* **Security ID (`type_id.go`):** The crucial, unambiguous link that decouples the user's personal, short-hand tickers from the global, canonical identifiers of financial assets.
* **Counterparty Account:** A new core concept representing the financial balance with a specific external entity (e.g., "Landlord", "John Doe", "ClientX").
* **Reporting Functions (e.g., `reports_holding.go`):** Stateless "brains" of the application. They are pure functions that take the **Ledger** as input and produce insights (e.g., holdings, gains, performance summaries) as output.

---
## 3. Components (The Code Modules)

This section describes the main Go packages and their distinct responsibilities.

* **`portfolio` (Core Logic):** This is the main library package containing the implementation of the Core Ontology (`ledger.go`, `transactions.go`, `type_id.go`) and the business logic (reporting functions). This package is completely decoupled from the user interface and data persistence layers.
* **`cmd` (User Interface):** This package implements the command-line interface. Each command is a thin wrapper that is responsible for parsing flags, validating user input, and calling the appropriate logic in the `portfolio` package to perform actions or calculations. It also contains the logic for discovering and executing external extension commands.
* **Core Business Types (within `portfolio` package):** These are the value-object types that represent fundamental business concepts. They are located in `types_*.go` files (e.g., `type_id.go`, `types_money.go`). This includes `ID`, `Money`, `Quantity`, `Date`, `Range`, and `Period`.
* **Persistence (within `portfolio` package):** This component is responsible for all I/O operations with the `transactions.jsonl` file. It ensures data is read and written in a canonical, backward-compatible way. It is located in `encode_ledger.go`.
    *   **Encoding (Writing):** To guarantee a stable, canonical output, each transaction type implements its own `MarshalJSON` method. These methods use a custom `jsonObjectWriter` to control the exact order and format of the JSON fields.
    *   **Decoding (Reading):** The `DecodeLedger` function (in `encode_ledger.go`) reads the `.jsonl` file line-by-line. It uses a two-pass decoding strategy: first, it identifies the transaction `command`, then it unmarshals the line into the corresponding concrete transaction struct.
* **Market Data Providers:** These components are responsible for fetching market data (prices, splits, dividends) from third-party sources. They are the only parts of the application permitted to make network requests and are invoked by the `fetch` command. There are two types:
    *   **Internal Providers (e.g., `eodhd`, `amundi`):** These are Go packages compiled directly into the `pcs` binary, offering built-in support for common data sources.
    *   **External Providers (Future):** Following the general extension mechanism, `pcs` will support external providers as standalone executables (e.g., `pcs-fetch-myprovider`). This will allow the community to add support for any data source without modifying the core application.

---
## 4. Data View (The Files)

This section describes the physical layout and format of the data stored on disk.

* **`transactions.jsonl`:** Stores the **Ledger**. It is an append-only file where each line is a single JSON object representing a transaction (including personal transactions like buys/sells and market data like prices/splits). For canonical representation, the file is sorted by date.

---
## 5. Documentation and Artifacts
 
This section outlines the key documents that guide and describe the project. These are divided into two categories: developer-focused strategic documents and user-focused, test-verified guides.
 
### 5.1. Developer Documentation
 
These documents define the "why" and "how" of the project's construction and are intended for contributors.
 
*   **`ARCHITECTURE.md` (This Document):** The technical blueprint. It describes the system's components, their responsibilities, and the principles that govern their interaction. It is the single source of truth for the project's design.
*   **`Vision.md`:** The product strategy document. It defines the core value proposition, target audience, and long-term goals, guiding all feature development.
 
### 5.2. User Documentation (Test-Verified)
 
These documents are for the end-users of `pcs`. A key principle is that all user-facing documentation containing command examples is **testable and automatically verified**. The integration tests in `docs/topics_test.go` execute the code blocks within these markdown files to ensure they are always accurate and up-to-date with the current implementation. When a feature is updated, the corresponding documentation and tests must be updated in lockstep.
 
*   **`README.md`:** The primary entry point for new users. It provides a high-level overview and a "Getting Started" tutorial.
*   **User Manual Topics (`docs/*.md`):** A collection of detailed, topic-specific guides that form the complete user manual. They are accessible via the `pcs topic` command. The `docs/readme.md` file serves as the index for these topics.

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