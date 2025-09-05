## Extending `pcs`

`pcs` is designed to be extensible, allowing you to add your own custom commands. This is particularly useful for integrating with proprietary bank APIs, importing data from custom formats, or adding specialized analysis tools.

The primary and most flexible way to add functionality is by creating external command executables. When you run a command that `pcs` doesn't recognize (e.g., `pcs my-importer`), it will search your system's `PATH` for an executable file named `pcs-my-importer`.

If found, `pcs` will execute that file, passing along any additional arguments. This allows you to write extensions in any programming language (Go, Python, Bash, etc.) and keep them completely separate from the main `pcs` codebase.

**How it Works:**

1.  Create an executable script or binary (e.g., `pcs-my-importer`).
2.  Make sure it's executable (`chmod +x pcs-my-importer`).
3.  Place it in a directory that is part of your system's `$PATH`.
4.  You can now run it as if it were a native command: `pcs my-importer --some-flag`.

This is the recommended approach for adding custom importers or integrations.

#### Accessing Configuration from Extensions

To allow your external commands to seamlessly integrate with the user's setup, `pcs` passes its current configuration to the extension via environment variables. Your script or program can read these variables to know which files to access and which settings to use.

The following environment variables are set:

* `PCS_MARKET_FILE`: The absolute path to the `market.jsonl` file currently in use.
* `PCS_LEDGER_FILE`: The absolute path to the `transactions.jsonl` file currently in use.
* `PCS_DEFAULT_CURRENCY`: The default currency (e.g., "EUR") set by the user.

By reading these variables, your extension can operate on the same data as the core `pcs` tool without needing the user to specify file paths or API keys again.