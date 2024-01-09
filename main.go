package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/alecthomas/kingpin/v2"
	"github.com/brianvoe/gofakeit/v6"
	_ "github.com/go-sql-driver/mysql"
)

var (
	version       = "0.0.0"
	commit        = "HEAD"
	date          = "2021-09-30T00:00:00Z"
	builtBy       = "unknown"
	app           = kingpin.New("faker", "A command-line tool to generate fake data and insert into database").Version(strings.Join([]string{version, commit, date, builtBy}, " "))
	debug         = app.Flag("debug", "Enable debug mode").Short('d').Bool()
	batchSizePtr  = app.Flag("batch-size", "Number of records to insert in a batch").Default("10000").Int()
	numWorkersPtr = app.Flag("num-workers", "Number of workers to generate fake data").Default("10").Int()
	numRecordsPtr = app.Flag("num-records", "Number of records to generate").Default("10000000").Int()

	generate = app.Command("generate", "Generate fake data")
)

func main() {
	kingpinMustParse := kingpin.MustParse(app.Parse(os.Args[1:]))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	loggingConfig := newLoggingConfig(os.Stdout, app.Name, *debug)

	if err := setupLogging(ctx, loggingConfig); err != nil {
		slog.ErrorContext(ctx, "Failed to setup logging", "error", err)
		return
	}

	slog.InfoContext(ctx, "start generating fake data")

	go handleSignals(ctx, cancel)

	// Handle errors from goroutines.
	errChan := make(chan error, 1)

	// Initialize gofakeit
	faker := gofakeit.NewCrypto()
	gofakeit.SetGlobalFaker(faker)

	db, err := connectDB(ctx, NewDBConfig())
	if err != nil {
		slog.ErrorContext(ctx, "faker", "Error opening database", err)
		return
	}
	defer db.Close()

	switch kingpinMustParse {
	case generate.FullCommand():
		generateData(ctx, db, errChan)
	}

	// Handle errors from goroutines.
	go func() {
		select {
		case err := <-errChan:
			slog.ErrorContext(ctx, "Error occurred in goroutines", "error", err)
			cancel()
		case <-ctx.Done():
		}
	}()

	slog.InfoContext(ctx, "finish generating fake data")
}

func handleSignals(ctx context.Context, cancel context.CancelFunc) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		slog.DebugContext(ctx, "received signal", "signal", sig)
		cancel()
		slog.InfoContext(ctx, "cancel generating fake data")
		os.Exit(1)
	}()
}

func generateData(ctx context.Context, db *sql.DB, errChan chan<- error) {
	batchSize := *batchSizePtr
	slog.DebugContext(ctx, "Initialize goroutines configurations", "batchSize", batchSize)
	numWorkers := *numWorkersPtr
	slog.DebugContext(ctx, "Initialize goroutines configurations", "numWorkers", numWorkers)
	numRecords := *numRecordsPtr
	slog.DebugContext(ctx, "Initialize goroutines configurations", "numRecords", numRecords)

	dataChan := make(chan DataItems, batchSize)
	var wg sync.WaitGroup

	// Launch worker goroutines
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(ctx, &wg, dataChan, numRecords/numWorkers, &PanelOrderItemCreator{}, errChan)
	}

	// Launch bulk insert goroutine
	go bulkInserter(ctx, db, dataChan, batchSize, errChan)

	wg.Wait()       // wait for worker goroutines to finish
	close(dataChan) // close channel to notify bulk insert goroutine to stop
}

func worker(ctx context.Context, wg *sync.WaitGroup, dataChan chan<- DataItems, numRecords int, creator DataItemCreator, errChan chan<- error) {
	defer wg.Done()
	for i := 0; i < numRecords; i++ {
		dataItems, err := creator.Create()
		if err != nil {
			errChan <- err
			return
		}

		select {
		case dataChan <- dataItems:
		case <-ctx.Done():
			return // return to stop worker goroutine
		}
	}
}

func bulkInserter(ctx context.Context, db *sql.DB, dataChan <-chan DataItems, batchSize int, errChan chan<- error) {
	panelOrderItemsBatch := make(PanelOrderItems, 0, batchSize)

	for {
		select {
		case items, ok := <-dataChan:
			if !ok {
				slog.DebugContext(ctx, "Channel is closed so insert remaining rows")
				if len(panelOrderItemsBatch) > 0 {
					processBatch(ctx, db, panelOrderItemsBatch, errChan)
				}
				return
			}

			for _, item := range items {
				switch v := item.(type) {
				case PanelOrderItems:
					panelOrderItemsBatch = append(panelOrderItemsBatch, v...)
					if len(panelOrderItemsBatch) == batchSize {
						processBatch(ctx, db, panelOrderItemsBatch, errChan)
						panelOrderItemsBatch = make(PanelOrderItems, 0, batchSize)
					}

				default:
					slog.ErrorContext(ctx, "Unknown type in batch", "type", fmt.Sprintf("%T", v))
					errChan <- fmt.Errorf("unknown type in batch: %T", v)
					return
				}
			}
		case <-ctx.Done():
			slog.DebugContext(ctx, "Close channel to notify worker goroutines to stop")
			return
		}
	}
}

func processBatch(ctx context.Context, db *sql.DB, items interface{}, errChan chan<- error) {
	slog.DebugContext(ctx, "Processing batch", "item", fmt.Sprintf("%T", items))
	switch v := items.(type) {
	case PanelOrderItems:
		slog.DebugContext(ctx, "Inserting PanelOrderItems", "numItems", len(v), "item", fmt.Sprintf("%T", v))
		if err := v.BulkInsert(ctx, db); err != nil {
			slog.ErrorContext(ctx, "Failed to bulk insert", "error", err)
		}
	default:
		slog.ErrorContext(ctx, "Unknown type in batch", "type", fmt.Sprintf("%T", v))
		errChan <- fmt.Errorf("unknown type in batch: %T", v)
		return
	}
}
