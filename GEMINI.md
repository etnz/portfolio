## Guidelines

*This document outlines the rules for interacting with me. All AIs must follow these guidelines.*

### The Corrector's Prerogative
When I provide code, you must modify it minimally to fulfill my request. Do not regenerate the entire file or overwrite my personal style. Sometimes, to help you I reject your suggestions and fix the file myself. Be careful to not overwrite my changes.

### The Singular Package Rule
Go package names must be singular (e.g., `transaction`).


### The Commit Message Rule
Commit messages must follow the seven rules from Chris Beams' post "[How to Write a Git Commit Message](https://chris.beams.io/posts/git-commit/)":
1. Separate subject from body with a blank line
2. Limit the subject line to 50 characters
3. Capitalize the subject line
4. Do not end the subject line with a period
5. Use the imperative mood in the subject line
6. Wrap the body at 72 characters
7. Use the body to explain what and why vs. how

Do not use Conventional Commits style. Do not use prefixes like `feat:`, `fix:`, etc.

You **must** ensure that commit messages are clear, descriptive and comprehensive.

Always reference the issue number in the commit message using Github style. Use automatic closing issues keywords when appropriate.

Gemini, here is a robust way of writing you messages:
```bash
git commit -F - <<EOF
> Your subject line here
>
> Your detailed, multi-line body goes here without any need for quotes
> or escaping special characters.
>
> - Even bullet points work perfectly.
>
> Fixes #123
> EOF
```

### The Master Rule
From time to time, I will ask you to regenerate these guidelines, including this introductory line.

### The Github CLI Rule

To interact with Github issues or pull requests, you can use the `gh` command line.

When reading an issue always read all the comments too.

**Recipe for multi-line comments:**
Use a heredoc with `-F -` and a custom delimiter (e.g., `EOD`):
```bash
gh issue comment <issue_number> -F - <<EOD
Your multi-line comment here.
It can span multiple lines.
EOD
```

### The Testing Rule

Run project's test often to ensure everything is working as expected. At least before any commit.

## Working with Issues

When I am asking you to work on an issue, please follow these steps:
- Read the issue carefully, including all comments.
- If you have any questions, ask me before starting to work on the issue.
- If the issue is not clear, ask me for clarification before starting to work on the issue
- If the issue is too broad, ask me to split it into smaller issues.
- Design a solution before starting to work on the issue. Share the design with me and ask for my approval before coding.
- When you have my approval, comment on the issue with the designed solution (favor rationale, design decisions, trade-offs, etc over code or how).
- If I LGTM the design, start working on the issue. Otherwise I'll ask you to update the design.
- When you finish the work, ask me to review the change before committing it.
- always reference the issue number in the commit message using Github style. Use automatic closing issues keywords when appropriate.




## Designing Command-Line Interfaces

### The Dynamic Default
Default values for date flags must be dynamically set to the current day within the `SetFlags` method.

### The No-Nonsense Synopsis
Synopsis strings for `subcommands` must start with a lowercase letter and must not end with a period.

### Command Rule
When design a new command or changes in an exisiting command, always consider existing flags and command names to avoid conflicts and to keep a consistent user experience.
