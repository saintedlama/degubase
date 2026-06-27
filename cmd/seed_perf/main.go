// cmd/seed populates a DeguBase database with synthetic data for UI performance testing.
//
// Usage:
//
//	go run ./cmd/seed [flags]
//
// Flags: --data-dir, --workspace, --tables, --columns, --rows
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/saintedlama/degubase/internal/identity"
	"github.com/saintedlama/degubase/internal/infrastructure/store"
	"github.com/saintedlama/degubase/internal/models"
	"github.com/saintedlama/degubase/internal/schema"
	"github.com/spf13/pflag"

	_ "modernc.org/sqlite"
)

func main() {
	dataDir := pflag.String("data-dir", "data", "path to data directory (same as server)")
	wsName := pflag.String("workspace", "Performance Test", "name of the workspace to create")
	numTables := pflag.Int("tables", 100, "number of tables to create")
	numColumns := pflag.Int("columns", 20, "number of columns per table")
	numRows := pflag.Int("rows", 1000, "number of rows per table")
	pflag.Parse()

	if err := os.MkdirAll(filepath.Join(*dataDir, "db"), 0o755); err != nil {
		log.Fatalf("create db dir: %v", err)
	}

	dbPath := filepath.Join(*dataDir, "db", "degubase.db")
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)", dbPath)

	// Init store (runs migrations).
	s, err := store.New(dsn)
	if err != nil {
		log.Fatalf("init store: %v", err)
	}
	db := s.DB()

	ident := identity.NewSQLite(db)
	schem := schema.NewSQLite(db)

	// Open a second raw connection for bulk row inserts inside explicit transactions.
	rawDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		log.Fatalf("open raw db: %v", err)
	}
	rawDB.SetMaxOpenConns(1)
	defer rawDB.Close()

	ctx := context.Background()

	fmt.Printf("Seeding: 1 workspace, %d tables, %d columns/table, %d rows/table\n",
		*numTables, *numColumns, *numRows)
	start := time.Now()

	ws, err := ident.CreateWorkspace(ctx, *wsName, "Synthetic dataset for UI performance testing", "")
	if err != nil {
		log.Fatalf("create workspace: %v", err)
	}
	fmt.Printf("  workspace %q created (id=%d)\n", ws.Name, ws.ID)

	colTypes := columnTypeRotation(*numColumns)

	for t := 0; t < *numTables; t++ {
		tableName := fmt.Sprintf("Table %03d", t+1)
		tbl, err := schem.CreateTable(ctx, ws.ID, tableName, fmt.Sprintf("Auto-generated table %d", t+1), "", "")
		if err != nil {
			log.Fatalf("create table %d: %v", t+1, err)
		}

		// Create columns and collect their IDs + types for row data generation.
		cols := make([]models.Column, 0, *numColumns)
		for c, ct := range colTypes {
			col, err := schem.CreateColumn(ctx, tbl.ID, columnName(ct, c), ct, columnOptions(ct))
			if err != nil {
				log.Fatalf("create column %d on table %d: %v", c, t+1, err)
			}
			cols = append(cols, *col)
		}

		if err := bulkInsertRows(ctx, rawDB, tbl.ID, cols, *numRows); err != nil {
			log.Fatalf("insert rows for table %d: %v", t+1, err)
		}

		fmt.Printf("\r  tables: %d/%d", t+1, *numTables)
	}

	fmt.Printf("\nDone in %s\n", time.Since(start).Round(time.Millisecond))
}

// bulkInsertRows inserts numRows rows into the given table inside a single transaction.
func bulkInsertRows(ctx context.Context, db *sql.DB, tableID int64, cols []models.Column, numRows int) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO rows (table_id, data, created_at, updated_at) VALUES (?, ?, ?, ?)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	rng := rand.New(rand.NewSource(tableID))
	now := time.Now().UTC().Format(time.RFC3339)

	for i := 0; i < numRows; i++ {
		data, err := json.Marshal(generateRowData(cols, i, rng))
		if err != nil {
			tx.Rollback()
			return err
		}
		if _, err := stmt.ExecContext(ctx, tableID, string(data), now, now); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// generateRowData returns a map[colID]value suitable for JSON-encoding as row data.
func generateRowData(cols []models.Column, rowIdx int, rng *rand.Rand) map[string]any {
	data := make(map[string]any, len(cols))
	for _, col := range cols {
		key := strconv.FormatInt(col.ID, 10)
		data[key] = fakeValue(col.Type, rowIdx, rng)
	}
	return data
}

func fakeValue(ct models.ColumnType, rowIdx int, rng *rand.Rand) any {
	switch ct {
	case models.ColumnTypeText:
		return randomWords(rng, 3, 6)
	case models.ColumnTypeLongText, models.ColumnTypeMarkdown:
		return randomSentences(rng, 2, 4)
	case models.ColumnTypeEmail:
		return fmt.Sprintf("user%d@example.com", rowIdx+1)
	case models.ColumnTypeURL:
		return fmt.Sprintf("https://example.com/item/%d", rowIdx+1)
	case models.ColumnTypeNumber, models.ColumnTypeRating:
		return rng.Intn(100) + 1
	case models.ColumnTypeCurrency:
		return float64(rng.Intn(100000)) / 100.0
	case models.ColumnTypePercent:
		return rng.Intn(101)
	case models.ColumnTypeDate:
		return randomDate(rng)
	case models.ColumnTypeDatetime:
		return randomDatetime(rng)
	case models.ColumnTypeCheckbox:
		return rng.Intn(2) == 1
	case models.ColumnTypeSingleSelect:
		opts := []string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon"}
		return opts[rng.Intn(len(opts))]
	case models.ColumnTypeMultiSelect:
		opts := []string{"Red", "Green", "Blue", "Yellow", "Purple"}
		n := rng.Intn(3) + 1
		rng.Shuffle(len(opts), func(i, j int) { opts[i], opts[j] = opts[j], opts[i] })
		return opts[:n]
	default:
		return nil
	}
}

// columnTypeRotation returns a slice of numColumns column types distributed
// across common types in a fixed proportion.
func columnTypeRotation(n int) []models.ColumnType {
	pool := []models.ColumnType{
		models.ColumnTypeText,
		models.ColumnTypeText,
		models.ColumnTypeNumber,
		models.ColumnTypeEmail,
		models.ColumnTypeCheckbox,
		models.ColumnTypeSingleSelect,
		models.ColumnTypeLongText,
		models.ColumnTypeURL,
		models.ColumnTypeDate,
		models.ColumnTypeCurrency,
	}
	types := make([]models.ColumnType, n)
	for i := range types {
		types[i] = pool[i%len(pool)]
	}
	return types
}

func columnName(ct models.ColumnType, idx int) string {
	base := strings.ReplaceAll(string(ct), "-", " ")
	return fmt.Sprintf("%s %d", base, idx+1)
}

func columnOptions(ct models.ColumnType) json.RawMessage {
	switch ct {
	case models.ColumnTypeSingleSelect, models.ColumnTypeMultiSelect:
		opts, _ := json.Marshal(map[string]any{
			"choices": []string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon"},
		})
		return opts
	}
	return nil
}

// ---- fake data helpers ----

var adjectives = []string{"quick", "lazy", "bright", "dark", "swift", "calm", "bold", "warm"}
var nouns = []string{"river", "mountain", "forest", "city", "ocean", "valley", "cloud", "bridge"}

func randomWords(rng *rand.Rand, min, max int) string {
	n := min + rng.Intn(max-min+1)
	words := make([]string, n)
	for i := range words {
		if i == 0 {
			w := adjectives[rng.Intn(len(adjectives))]
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		} else if i%2 == 0 {
			words[i] = adjectives[rng.Intn(len(adjectives))]
		} else {
			words[i] = nouns[rng.Intn(len(nouns))]
		}
	}
	return strings.Join(words, " ")
}

func randomSentences(rng *rand.Rand, min, max int) string {
	n := min + rng.Intn(max-min+1)
	sentences := make([]string, n)
	for i := range sentences {
		sentences[i] = randomWords(rng, 5, 10) + "."
	}
	return strings.Join(sentences, " ")
}

func randomDate(rng *rand.Rand) string {
	base := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	days := rng.Intn(365 * 5)
	return base.AddDate(0, 0, days).Format("2006-01-02")
}

func randomDatetime(rng *rand.Rand) string {
	base := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	secs := rng.Intn(365 * 5 * 24 * 3600)
	return base.Add(time.Duration(secs) * time.Second).Format(time.RFC3339)
}
