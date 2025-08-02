package security

// This file contains functions to update the database with latest prices.
// updatable are:
// - security available at EODHD. If the security has currently no data, a default starting date is used.
// - forex pairs.
//
// In order to figure that out, the security.ID is used to identify the security ISIN and MIC where applicable,
// or the forex pair.
// there might be other thypes of securities, but they are not supported by update, yet (like privately traded assets)
