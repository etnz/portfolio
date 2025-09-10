# pcs Architecture and Guiding Principles

This document outlines the architecture of the `pcs` portfolio tool. It serves as the single source of truth for the project's design, conventions, and the relationship between its components. The primary purpose of this document is to ensure that all development, whether by human or AI contributors, adheres to a consistent and coherent vision, promoting long-term maintainability and clarity.

---
## 1. Architectural Goals

The design of `pcs` is guided by a set of core principles that inform all development decisions.

* **Local-First & Private:** All user data is stored on the local filesystem. The tool must be fully functional without requiring access to cloud services for its core operations.
* **Auditable & Version-Controllable:** Data is stored in human-readable, line-oriented text files (`.jsonl`) to work seamlessly with version control systems like Git. This ensures that every change to a portfolio is transparent and auditable.
* **Extensible:** The system is designed to be extended with custom, external commands (e.g., `pcs-my-importer`) to support new data sources and integrations without modifying the core application.
* **Single Source of Truth:** The Go source code is the canonical source for all user-facing documentation. The user manual and other artifacts are derived from the code itself to prevent documentation drift.

---
## 2. Core Ontology (The Language)

This section defines the key conceptual entities that form the domain language of the portfolio.

* **Ledger (`ledger.go`, `transactions.go`):** The immutable, chronological record of all user-initiated actions (buys, sells, deposits). It represents the user's input and financial history.
* **Market Data (`market.go`):** A database of security information and their historical prices. It represents data from the external financial world.
* **Security ID (`security.go`):** The crucial, unambiguous link between the **Ledger** and **Market Data**. It decouples the user's personal, short-hand tickers from the global, canonical identifiers of financial assets.
* **Counterparty Account:** A new core concept representing the financial balance with a specific external entity (e.g., "Landlord", "John Doe", "ClientX").
* **Accounting System (`accounting.go`):** The stateless "brain" of the application. It is a pure function that takes the **Ledger** and **Market Data** as input and produces insights (e.g., holdings, gains, performance summaries) as output.

---
## 3. Components (The Code Modules)

This section describes the main Go packages and their distinct responsibilities.

* **`portfolio` (Core Logic):** This is the main library package containing the implementation of the Core Ontology (`ledger.go`, `market.go`, `security.go`) and the business logic (`accounting.go`). This package is completely decoupled from the user interface and data persistence layers.
* **`cmd` (User Interface):** This package implements the command-line interface. Each command is a thin wrapper that is responsible for parsing flags, validating user input, and calling the appropriate logic in the `portfolio` package to perform actions or calculations. It also contains the logic for discovering and executing external extension commands.
* **Persistence (within `portfolio` package):** This component is responsible for all I/O operations. It handles the encoding and decoding of the Ledger and Market Data to and from the `.jsonl` file format. It ensures data is read and written in a canonical, backward-compatible way. Import/export logic for specific formats (e.g., Amundi) is handled by dedicated commands in the `cmd` package.
* **External Data Sources (within `portfolio` package):** These components (e.g., `eodhd.go`) are responsible for fetching data from third-party APIs. They are the only parts of the application that are permitted to make network requests.

---
## 4. Data View (The Files)

This section describes the physical layout and format of the data stored on disk.

* **`transactions.jsonl`:** Stores the **Ledger**. It is an append-only file where each line is a single JSON object representing a transaction. For canonical representation, the file is sorted by date.
* **`market.jsonl`:** Stores the **Market Data**, including security definitions and their complete price and split histories. Each line represents a distinct piece of market data.

---
## 5. Documentation and Artifacts

This defines the relationship between the code and the documentation, enforcing the "Single Source of Truth" principle.

* **User Manual (`docs/readme.md`):** The primary user-facing documentation, which serves as an index for all documentation topics. Its "Command Reference" section **must be derived directly** from the Go source code of the commands to ensure accuracy.
* **README (`README.md`):** A high-level introduction and getting-started guide for new users.
* **Testable Documentation (`docs/topics_test.go`):** The examples in `README.md` and all topic files inside `docs/` are not merely illustrative; they are part of an integration test that must always pass, ensuring the primary documentation is always correct.

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
3.  **Responsibility:** The developer is responsible for updating all relevant artifacts as defined by this document (code, tests, `UserManual.md`, `README.md`, etc.).