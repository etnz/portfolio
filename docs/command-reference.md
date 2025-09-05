## Command Reference

`pcs` commands follow a consistent structure: `pcs [global flags] <subcommand> [subcommand flags] [arguments]`.

### Subcommands

`pcs` organizes its functionalities into subcommands, grouped by their purpose. You can see a list of all available subcommands and their groups by running `pcs help`.

### Global Flags

These flags can be used with any `pcs` command and affect the overall behavior of the application. They are typically placed directly after `pcs` and before the subcommand. For a detailed explanation of each global flag, refer to `pcs topic global-flags`.

### Command-Specific Flags

Many subcommands have their own flags to control their specific behavior. These flags are placed after the subcommand.

#### Common Flags:

*   `-d <date>`: Specifies the effective date for a command, crucial for historical reports and time-sensitive calculations. `pcs` supports flexible date formats, including relative dates (e.g., `-1d` for yesterday) and partial dates (e.g., `08-29` for August 29th of the current year). For more details, see `pcs topic dates`.

    **Example:**
    ```bash
    pcs daily -d 2023-01-15
    pcs holding -d -1w
    ```

*   `-c <currency>`: Sets the currency for a command's financial calculations and output. The currency should be specified using its ISO 4217 code (e.g., `USD`, `EUR`, `JPY`).
*   `-s <security>`: Identifies a security by its user-defined ticker, primarily used in ledger-related commands to record transactions against a specific holding.
*   `-id <security-id>`: Identifies a security by its globally unique ID (e.g., ISIN.MIC), mainly used in market-data related commands to fetch or update security information.
*   `-q <quantity>`: Specifies the number of shares or units for a transaction. It is always a positive value.
*   `-a <amount>`: Defines the total monetary value of a transaction, such as the total cost of a purchase or the total proceeds from a sale. It is always a positive value.
*   `-m <memo>`: Adds a descriptive note or comment to a transaction for future reference.
*   `-u`: Attempts an update of intraday prices from external providers before generating a report, ensuring the most current data is used.
