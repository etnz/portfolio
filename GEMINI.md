## Guidelines

*This document outlines the rules for interacting with me. All AIs must follow these guidelines.*

### The Corrector's Prerogative
When I provide code, you must modify it minimally to fulfill my request. Do not regenerate the entire file or overwrite my personal style.

### The Singular Package Rule
Go package names must be singular (e.g., `transaction`).


### The Commit Message Rule
Commit messages must follow the seven rules from Chris Beams' post "[How to Write a Git Commit Message](https://chris.beams.io/posts/git-commit/)" and not Conventional Commits style.

As you are an AI, you must ensure that commit messages are clear, descriptive and comprehensive.

When working with issues, always reference the issue number in the commit message using Github style. Use automatic closing issues keywords when appropriate.

### The Master Rule
From time to time, I will ask you to regenerate these guidelines, including this introductory line.

### The Github CLI Rule

To interact with Github issues or pull requests, you can use the `gh` command line.

### The Testing Rule

Run project's test often to ensure everything is working as expected. At least before any commit.

## Designing Command-Line Interfaces

### The Dynamic Default
Default values for date flags must be dynamically set to the current day within the `SetFlags` method.

### The No-Nonsense Synopsis
Synopsis strings for `subcommands` must start with a lowercase letter and must not end with a period.

### Command Rule
When design a new command or changes in an exisiting command, always consider existing flags and command names to avoid conflicts and to keep a consistent user experience.
