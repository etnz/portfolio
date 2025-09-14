## Extending `pcs`

### Purpose

The extension mechanism is a core feature designed to support the pillars of **unification** and **durability** (through effortless tracking). It allows you to connect `pcs` to any data source—from proprietary bank APIs to custom file formats—ensuring that you can bring your entire financial world into one place with minimal friction.

### How It Works

The primary and most flexible way to add functionality is by creating external command executables. When you run a command that `pcs` doesn't recognize (e.g., `pcs my-importer`), it will search your system's `PATH` for an executable file named `pcs-my-importer`.

If found, `pcs` will execute that file, passing along any additional arguments. This allows you to write extensions in any programming language (Go, Python, Bash, etc.) and keep them completely separate from the main `pcs` codebase.

**Example Workflow:**
1.  You create an executable script named `pcs-my-importer`.
2.  You make it executable (`chmod +x pcs-my-importer`).
3.  You place it in a directory that is part of your system's `$PATH`.
4.  You can now run it as if it were a native `pcs` command:
    `pcs my-importer --some-flag`

#### Accessing Configuration from Extensions

To allow your external commands to seamlessly integrate with the user's setup, `pcs` passes its current configuration to the extension via environment variables. Your script or program can read these variables to know which files to access and which settings to use.

The following environment variables are set:

* `PCS_LEDGER_FILE`: The absolute path to the `transactions.jsonl` file currently in use.
* `PCS_DEFAULT_CURRENCY`: The default currency (e.g., "EUR") set by the user.

By reading these variables, your extension can operate on the same data as the core `pcs` tool without needing the user to specify file paths or API keys again.