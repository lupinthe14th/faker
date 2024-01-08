package main

import (
	"context"
	"database/sql"
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

	dataChan := make(chan PanelOrderItems, batchSize)
	var wg sync.WaitGroup

	// Launch worker goroutines
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(ctx, &wg, dataChan, numRecords/numWorkers, errChan)
	}

	// Launch bulk insert goroutine
	go bulkInserter(ctx, db, dataChan, batchSize)

	wg.Wait()       // wait for worker goroutines to finish
	close(dataChan) // close channel to notify bulk insert goroutine to stop
}

func worker(ctx context.Context, wg *sync.WaitGroup, dataChan chan<- PanelOrderItems, numRecords int, errChan chan<- error) {
	defer wg.Done()
	for i := 0; i < numRecords; i++ {
		panelOrderItem := PanelOrderItem{}
		if err := gofakeit.Struct(&panelOrderItem); err != nil {
			errChan <- err
			return
		}

		select {
		case dataChan <- PanelOrderItems{panelOrderItem}: // send to channel
		case <-ctx.Done():
			return // return to stop worker goroutine
		}
	}
}

func bulkInserter(ctx context.Context, db *sql.DB, dataChan <-chan PanelOrderItems, batchSize int) {
	bulkInsBatch := make(PanelOrderItems, 0, batchSize)
	for {
		select {
		case panelOrderItem, ok := <-dataChan:
			if !ok {
				slog.DebugContext(ctx, "Channel is closed so insert remaining rows")
				if len(bulkInsBatch) > 0 {
					if err := bulkInsBatch.BulkInsert(ctx, db); err != nil {
						slog.ErrorContext(ctx, "Failed to bulk insert", "error", err)
					}
				}
				return
			}

			bulkInsBatch = append(bulkInsBatch, panelOrderItem...)
			if len(bulkInsBatch) == batchSize {
				if err := bulkInsBatch.BulkInsert(ctx, db); err != nil {
					slog.ErrorContext(ctx, "Failed to bulk insert", "error", err)
					return
				}
				bulkInsBatch = make(PanelOrderItems, 0, batchSize)
			}
		case <-ctx.Done():
			slog.DebugContext(ctx, "Close channel to notify worker goroutines to stop")
			return
		}
	}
}
