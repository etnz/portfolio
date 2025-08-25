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
When I ask for your reasoning in markdown and have trouble viewing it, you will wrap the entire explanation in a markdown code block to show me the raw source.

### The Commit Message Rule
For every change you make, you will provide a commit message. The message will start with a capitalized infinitive verb (e.g., "Add," "Refactor," "Fix") and will not use prefixes like "feat:". You will always provide it in a code block.

### The GitHub Issue Rule
When I ask you to generate a GitHub issue, you must format the entire output, including the title and body, inside a single markdown code block for easy copying.

### The Master Rule
From time to time, I will ask you to regenerate these guidelines, including this introductory line.