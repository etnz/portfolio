// Package transaction provides a set of functions and types to manage
// transactions in a financial system. 
//   * `open     -on <day> -s <security> -m <rationale> -q <number of shares> -p <share price>` creates a new position for the given security
//   * `add      -on <day> -s <security> -m <rationale> -q <number of shares> -p <share price>` add to an existing position for the given security
//   * `trim     -on <day> -s <security> -m <rationale> -q <number of shares> -p <share price>` trime an existing position for the given security
//   * `close    -on <day> -s <security> -m <rationale> -p <share price>` close an existing position for the given security
//   * `dividend -on <day> -s <security> -m <rationale> -a <amount paid>` record dividend pay out
//   * `deposit  -on <day> -m <rationale> -a <amount> -c <currency>` deposit cash into the portfolio's cash account.
//   * `withdraw -on <day> -m <rationale> -a <amount> -c <currency>` withdraw cash from the portfolio's cash account.
//   * operations like `add` or `open` will withdraw the proceeds from the cash account and fail if the account is not provisioned.
//   * similarly operations like `trim` or `close` will deposit the proceeds to the cash account.
//   * Transactions are stored in a jsonl file per year, in a format specific to each instruction.
//
//
package transaction

