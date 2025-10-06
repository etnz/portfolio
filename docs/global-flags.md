## Global Flags

`pcs` uses global flags to modify the default behavior for all subcommands. This allows for flexible configuration without changing files.

The primary use cases for these flags are:

1.  **Specifying Data Files:** You can direct `pcs` to use different ledgers. This is useful for managing multiple distinct portfolios.
2.  **Managing Secrets:** You can specify API keys and other secrets for data providers via environment variables.

### Portfolio Path (The Modern Way)

The future way to manage your data is to use the `--portfolio` flag or the `PORTFOLIO_PATH` environment variable. This points `pcs` to a directory containing one or more ledger files (ending in `.jsonl`).

This approach is the future of `pcs` and enables multi-ledger support, allowing you to organize your finances into separate files (e.g., `personal.jsonl`, `family/joint.jsonl`) within a single portfolio directory.

### Ledger File (Legacy)

The `--ledger-file` flag specifies a single ledger file. It is considered legacy and will be phased out in the future. It is still the 
only way to access all features.

To see a complete and up-to-date list of all available global flags and their descriptions, run the help command:

```bash
pcs flags
```