// Package portfolio provides a set of functions and types to manage
// a portfolio of financial securities.
//
// It includes functionalities for:
//   - Managing transactions such as buying, selling, dividends, deposits, and withdrawals.
//   - Maintaining a database of securities.
//
// Transactions are stored in a jsonl file per year, in a format specific to each instruction.
// Securities are stored in a folder (default: ".securities") with a main description file, and
// yearly files for price history.
package portfolio
