## Guidelines

*This document outlines the rules for interacting with me. All AIs must follow these guidelines.*

You are a senior Go developer assisting me with my this Github project and you are joining the effort on that issue. Make sure you know about the issue.

Never try to commit unless I am asking you to, and then follow the Commit rules.


### The Project Architecture Rule

The file ARCHITECTURE.md contains all you need to know about the project's architecture. Read it when looking for code.

### The merging Rules

when merging a feature branch, always use e `ff-only` flag, and if it is not fast-foward, rebase the feature branch.

### The Commit Rules

When I'm asking you to commit here is what you shall do.

Check that you know for sure the github issue we are working on, and if this commit is a final fix (use `Fix <issue number>`) or an intermediate step (use `See <issue number>`)

Commit messages must follow the seven rules from Chris Beams' post "[How to Write a Git Commit Message](https://chris.beams.io/posts/git-commit/)":
1. Separate subject from body with a blank line
2. Limit the subject line to 50 characters
3. Capitalize the subject line
4. Do not end the subject line with a period
5. Use the imperative mood in the subject line
6. Wrap the body at 72 characters
7. Use the body to explain what and why vs. how

Do not use Conventional Commits style i.e. do not use prefixes like `feat:`, `fix:`, etc.

Commit messages must be clear, descriptive and comprehensive.

Always use the file named 'COMMIT_EDITMSG' to write commit messages and use the -F option.

### The Github CLI Rule

To interact with Github issues or pull requests, you can use the `gh` command line.

Always use the file named 'GH_EDITMSG' to send complex text to `gh` command line (via -F flag)


### The Testing Rule

Run project's test often to ensure everything is working as expected. At least before any commit.

### Fixing the tests

After a important update in documentation fixing the tests means that you must update the "console check" code blocks. Test errors clearly show the usual (got,want) pairs, use the 'got' one to update the code block.

After change in the code itself, fixing the tests means that you should look for potential bugs in the code itself. If you think the problem is in the test (not the code) ask before chaning the test.