## Extending `pcs`

### Purpose

The extension mechanism is a core feature designed to support the pillars of **unification** and **durability** (through effortless tracking). It allows you to connect `pcs` to any data source—from proprietary bank APIs to custom file formats—ensuring that you can bring your entire financial world into one place with minimal friction.

This is especially powerful for creating new **market data providers**. While `pcs` includes built-in providers like `eodhd` and `insee`, you can easily add your own.

### How It Works: External Commands

The primary way to add functionality is by creating external command executables. When you run a command that `pcs` doesn't recognize as built-in (e.g., `pcs my-provider`), it will search your system's `PATH` for an executable file named `pcs-my-provider`.

If found, `pcs` will execute that file, passing along any additional arguments. This allows you to write extensions in any programming language (Go, Python, Bash, etc.) and keep them completely separate from the main `pcs` codebase.

### Example: Creating a New Provider

Let's say you want to create a new provider called `my-bank`.

1.  You create an executable script named `pcs-my-importer`.
2.  You make it executable (`chmod +x pcs-my-importer`).
3.  You place it in a directory that is part of your system's `$PATH`.
4.  You can now run it as if it were a native `pcs` command:
    `pcs my-importer --some-flag`

#### Accessing Configuration from Extensions

To allow your external commands to seamlessly integrate with the user's setup, `pcs` passes its current configuration to the extension via environment variables. Your script or program can read these variables to know which files to access and which settings to use.

The following environment variables are set:

* `PORTFOLIO_PATH`: The absolute path to the portfolio directory currently in use. This directory can contain multiple ledger files (`.jsonl`).
* `PCS_DEFAULT_CURRENCY`: The default currency (e.g., "EUR") set by the user.

By reading these variables, your extension can operate on the same data as the core `pcs` tool without needing the user to specify file paths or API keys again.