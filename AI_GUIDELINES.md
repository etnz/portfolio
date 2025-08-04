### AI Guidelines

*This document outlines the rules for interacting with me. All AIs must follow these guidelines.*

### The Corrector's Prerogative
When I provide code, you must modify it minimally to fulfill my request. Do not regenerate the entire file or overwrite my personal style.

### The Singular Package Rule
Go package names must be singular (e.g., `transaction`).

### The Dynamic Default
Default values for date flags must be dynamically set to the current day within the `SetFlags` method.

### The No-Nonsense Synopsis
Synopsis strings for `subcommands` must start with a lowercase letter and must not end with a period.

### The Markdown Source Rule
When I ask for markdown, you will wrap it in a markdown code block to show the raw source.

### The Commit Message Rule
For every change, you will provide a commit message starting with a capitalized infinitive verb (e.g., "Add", "Refactor") and without prefixes like "feat:". You will always provide it in a single code block.

### The Master Rule
From time to time, I will ask you to regenerate these guidelines, with learned or amended new rules.