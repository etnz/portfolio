package cmd

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/etnz/portfolio"
	"github.com/etnz/portfolio/date"
	"github.com/google/subcommands"
)

// --- Structs to unmarshal the Amundi JSON file ---

// AmundiTransactionFile holds the top-level structure of the JSON file.
type AmundiTransactionFile struct {
	Operations []AmundiOperation `json:"operationsIndividuelles"`
}

// AmundiOperation represents a single high-level operation, which can contain multiple instructions.
type AmundiOperation struct {
	Memo         string              `json:"libelleCommunication"`
	Id           string              `json:"idOpeInd"`
	Type         string              `json:"type"`
	DateDemand   date.Date           `json:"dateDeLaDemande"`
	DateCompta   date.Date           `json:"dateComptabilisation"`
	Instructions []AmundiInstruction `json:"instructions"`
	Reglements   []AmundiReglement   `json:"reglements"`
}

// AmundiInstruction is a detailed leg of a transaction, like a specific buy or sell.
type AmundiInstruction struct {
	Type       string    `json:"type"`                // e.g., "ARB", "RACH_TIT", "SOUS_MTT"
	Id         string    `json:"idInstruction"`       // e.g.,
	Statut     string    `json:"statut"`              // e.g: "ANNULE",
	Indicator  string    `json:"indicateurArbitrage"` // "Source" or "Cible" for ARB
	DateVL     date.Date `json:"dateVlReel"`
	Price      float64   `json:"vlReel"`
	Quantity   float64   `json:"nombreDeParts"`
	Security   string    `json:"codeFonds"`
	FundName   string    `json:"nomFonds"`
	Dispositif string    `json:"libelleDispositifMetier"`
	Amount     float64   `json:"montantNet"`
}

// AmundiReglement represents a cash settlement, like a withdrawal.
type AmundiReglement struct {
	Type   string  `json:"type"` // e.g., "VIROUT"
	Amount float64 `json:"montant"`
}

// --- import-amundi Command ---
// this url to get some saving accounts (worker saving)
// https://epargnant.amundi-ee.com/api/individu/operations?metier=ESR&flagFiltrageWebSalarie=true&flagInfoOC=Y&filtreStatutModeExclusion=false&flagRu=true&offset=0&limit=100
// this one for the other type (assurance)
// https://epargnant.amundi-ee.com/api/individu/operations?metier=ASSU&offset=0&flagFiltrageWebSalarie=true&flagInfoOC=Y&filtreStatutModeExclusion=false&flagRu=true&limit=100

type importAmundiCmd struct{}

func (*importAmundiCmd) Name() string { return "import-amundi" }
func (*importAmundiCmd) Synopsis() string {
	return "Converts an Amundi transactions JSON file to JSONL format."
}
func (*importAmundiCmd) Usage() string {
	return `pcs import-amundi <amundi_transactions.json>


  Reads Amundi's JSON file for transactions and outputs transactions in the standard JSONL format to stdout.
  Example: pcs import-amundi /path/to/amundi_transactions.json > transactions.jsonl

  Translation cannot be perfect, use with care and review the output.
`
}

func (c *importAmundiCmd) SetFlags(f *flag.FlagSet) {}

func (c *importAmundiCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "Please provide the path to the amundi_transactions.json file.")
		return subcommands.ExitUsageError
	}
	filePath := f.Arg(0)

	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", filePath, err)
		return subcommands.ExitFailure
	}

	var amundiFile AmundiTransactionFile
	if err := json.Unmarshal(data, &amundiFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON file: %v\n", err)
		return subcommands.ExitFailure
	}

	ledger := portfolio.NewLedger()

	// there might be repetitions in the operations (due to paging in API)
	operationsDone := make(map[string]struct{})

	for _, op := range amundiFile.Operations {
		if _, done := operationsDone[op.Id]; done {
			// skip already processed operations
			continue
		}
		operationsDone[op.Id] = struct{}{} // mark as treated
		txs, err := parseAmundiOperation(op)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: skipping operation (memo: %q): %v\n", op.Memo, err)
			continue
		}
		ledger.Append(txs...)
	}

	if err := portfolio.EncodeLedger(os.Stdout, ledger); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding transactions to JSONL: %v\n", err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}

// parseAmundiOperation converts a single Amundi operation into a slice of standard transactions.
func parseAmundiOperation(op AmundiOperation) ([]portfolio.Transaction, error) {
	transactions := make([]portfolio.Transaction, 0)
	memo := op.Memo
	currency := "EUR" // Currency is consistently EUR in the provided data.

	instructionIds := make(map[string]struct{}) // track instruction that have been processed.

	switch op.Type {
	case "ARB": // Arbitrage, Transfert, Réallocation
		for _, inst := range op.Instructions {
			memo := inst.Dispositif + ": " + memo + ": " + inst.FundName
			if inst.Statut == "ANNULE" {
				log.Println("skip cancelled instruction", memo)
				instructionIds[inst.Id] = struct{}{}
				continue
			}
			if inst.Indicator == "Source" {
				instructionIds[inst.Id] = struct{}{}
				sellTx := portfolio.NewSell(inst.DateVL, memo, inst.Security, inst.Quantity, inst.Price)
				transactions = append(transactions, sellTx)
			} else if inst.Indicator == "Cible" {
				instructionIds[inst.Id] = struct{}{}
				buyTx := portfolio.NewBuy(inst.DateVL, memo, inst.Security, inst.Quantity, inst.Price)
				transactions = append(transactions, buyTx)
			}
		}

	case "RACH_HE": // Remboursement, Frais de tenue de compte
		for _, inst := range op.Instructions {
			if inst.Type == "RACH_TIT" {
				instructionIds[inst.Id] = struct{}{}
				memo := strings.Join([]string{inst.Id, inst.Dispositif, memo, inst.FundName}, ": ")
				sellTx := portfolio.NewSell(inst.DateVL, memo, inst.Security, inst.Quantity, inst.Price)
				// record the associated withdrawal
				withdrawTx := portfolio.NewWithdraw(inst.DateVL, memo, currency, inst.Amount)
				transactions = append(transactions, sellTx, withdrawTx)

			}
		}

	case "SOUS": // Versement, Participation, Intéressement
		var totalAmount float64

		for _, inst := range op.Instructions {
			memo := strings.Join([]string{inst.Id, inst.Dispositif, memo, inst.FundName}, ": ")

			if inst.Statut == "ANNULE" {
				log.Println("skip cancelled instruction", memo)
				instructionIds[inst.Id] = struct{}{}
				continue
			}

			if inst.Type == "SOUS_MTT" {
				instructionIds[inst.Id] = struct{}{}
				totalAmount += inst.Amount // accumulate buy amounts to create a global deposit after
				buyTx := portfolio.NewBuy(inst.DateVL, memo, inst.Security, inst.Quantity, inst.Price)
				transactions = append(transactions, buyTx)
			}
		}

		if totalAmount > 0 {
			depositTx := portfolio.NewDeposit(op.DateDemand, memo, currency, totalAmount)
			transactions = append(transactions, depositTx)
		}
	case "TRSF": // Arbitrage, Transfert, Réallocation
		var totalAmount float64
		for _, inst := range op.Instructions {
			memo := inst.Dispositif + ": " + memo + ": " + inst.FundName
			if inst.Statut == "ANNULE" {
				log.Println("skip cancelled instruction", memo)
				instructionIds[inst.Id] = struct{}{}
				continue
			}
			if inst.Indicator == "Source" {
				instructionIds[inst.Id] = struct{}{}
				totalAmount += inst.Amount
			} else if inst.Indicator == "Cible" {
				instructionIds[inst.Id] = struct{}{}
				buyTx := portfolio.NewBuy(inst.DateVL, memo, inst.Security, inst.Quantity, inst.Price)
				transactions = append(transactions, buyTx)
			}
		}
		if totalAmount > 0 {
			depositTx := portfolio.NewDeposit(op.DateDemand, memo, currency, totalAmount)
			transactions = append(transactions, depositTx)
		}

	default:
		return nil, fmt.Errorf("unhandled operation type: %q", op.Type)
	}

	// check that all instructions have been used
	for _, inst := range op.Instructions {
		if _, ok := instructionIds[inst.Id]; !ok {
			log.Println("instruction has not been used:", inst.Id)
		}
	}

	return transactions, nil
}
