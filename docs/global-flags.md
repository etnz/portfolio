## Global Flags

`pcs` uses global flags to modify the default behavior for all subcommands. This allows for flexible configuration without changing files.

Two primary use cases for these flags are:

1.  **Specifying Data Files:** You can direct `pcs` to use different ledger. This is useful for managing multiple distinct portfolios.
2.  **Managing Secrets:** You can specify API keys and other secrets for data providers.

To see a complete and up-to-date list of all available global flags and their descriptions, run the help command:

```bash
pcs flags
```