// Package portfolio provides a comprehensive set of functions and types for
// managing a personal financial portfolio. It is designed to be local-first,
// auditable, and extensible, ensuring users have full control and transparency
// over their financial data.
//
// The core functionalities include:
//   - Ledger Management: Recording and tracking all financial transactions
//     (e.g., buys, sells, dividends, deposits, withdrawals, currency conversions,
//     and accruals) in an immutable, chronological record.
//   - Market Data Integration: Storing and utilizing security information and
//     historical prices to provide accurate valuations and performance metrics.
//   - Accounting System: A stateless engine that processes ledger and market
//     data to generate insights such as holdings, gains, and performance summaries.
//   - Security Identification: A robust system for linking user-defined tickers
//     to globally unique security identifiers.
//   - Data Persistence: Handling the encoding and decoding of financial data
//     to and from human-readable, version-controllable formats (e.g., JSONL).
//
// This package serves as the foundational logic for the `pcs` command-line
// tool, ensuring that all operations are consistent and based on a single
// source of truth.
package portfolio
