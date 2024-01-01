package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/alecthomas/kingpin/v2"
	"github.com/brianvoe/gofakeit/v6"
	_ "github.com/go-sql-driver/mysql"
)

const (
	version = "0.0.0"
)

var (
	revision      = "HEAD"
	app           = kingpin.New("faker", "A command-line tool to generate fake data and insert into database").Version(version)
	debug         = app.Flag("debug", "Enable debug mode").Short('d').Bool()
	batchSizePtr  = app.Flag("batch-size", "Number of records to insert in a batch").Default("10000").Int()
	numWorkersPtr = app.Flag("num-workers", "Number of workers to generate fake data").Default("10").Int()
	numRecordsPtr = app.Flag("num-records", "Number of records to generate").Default("10000000").Int()

	generate = app.Command("generate", "Generate fake data")
)

func main() {
	kingpinMustParse := kingpin.MustParse(app.Parse(os.Args[1:]))

	logLevel := new(slog.LevelVar)
	opts := &slog.HandlerOptions{
		AddSource: *debug,
		Level:     logLevel,
	}
	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler).With(
		"revision", revision,
		"version", version,
		"app", app.Name,
	)

	ctx, cancel := context.WithCancel(context.Background())
	logger.InfoContext(ctx, "start generating fake data")

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	faker := gofakeit.NewCrypto()
	gofakeit.SetGlobalFaker(faker)

	db, err := connectDB(ctx, NewDBConfig())
	if err != nil {
		logger.ErrorContext(ctx, "faker", "Error opening database", err)
		os.Exit(1)
	}
	defer db.Close()
	if *debug {
		logLevel.Set(slog.LevelDebug)
	}

	batchSize := *batchSizePtr
	logger.DebugContext(ctx, "Initialize goroutines configurations", "batchSize", batchSize)
	numWorkers := *numWorkersPtr
	logger.DebugContext(ctx, "Initialize goroutines configurations", "numWorkers", numWorkers)
	numRecords := *numRecordsPtr
	logger.DebugContext(ctx, "Initialize goroutines configurations", "numRecords", numRecords)

	switch kingpinMustParse {
	case generate.FullCommand():
		panelOrderItemsChan := make(chan PanelOrderItems, batchSize)
		var wg sync.WaitGroup

		// Worker goroutines
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < numRecords/numWorkers; j++ {
					panelOrderItem := PanelOrderItem{}
					if err := gofakeit.Struct(&panelOrderItem); err != nil {
						logger.ErrorContext(ctx, "Failed to generate fake data", "error", err)
						continue
					}

					panelOrderItemsChan <- PanelOrderItems{
						panelOrderItem,
					}
				}
			}()
		}

		var bulkInsWg sync.WaitGroup
		bulkInsWg.Add(1) // for bulk insert goroutine

		// Bulk insert goroutine
		go func(ctx context.Context) {
			defer bulkInsWg.Done() // notify when bulk insert goroutine is done

			bulkInsBatch := make(PanelOrderItems, 0, batchSize)
			for {
				select {
				case panelOrderItem, ok := <-panelOrderItemsChan:
					if !ok {
						logger.DebugContext(ctx, "Channel is closed so insert remaining rows")
						if len(bulkInsBatch) > 0 {
							c, err := bulkInsBatch.BulkInsert(ctx, db)
							if err != nil {
								logger.ErrorContext(ctx, "Failed to bulk insert", "error", err)
							}
							logger.DebugContext(ctx, "bulk insert rows", "rows", c)
						}
						return
					}

					bulkInsBatch = append(bulkInsBatch, panelOrderItem...)
					if len(bulkInsBatch) == batchSize {
						c, err := bulkInsBatch.BulkInsert(ctx, db)
						if err != nil {
							logger.ErrorContext(ctx, "Failed to bulk insert", "error", err)
						}
						logger.DebugContext(ctx, "bulk insert rows", "rows", c)
						bulkInsBatch = make(PanelOrderItems, 0, batchSize)
					}
				case <-ctx.Done():
					logger.DebugContext(ctx, "Close channel to notify worker goroutines to stop")
					return
				}
			}
		}(ctx)

		wg.Wait() // wait for worker goroutines to finish
		close(panelOrderItemsChan)
		bulkInsWg.Wait() // wait for bulk insert goroutine to finish
	}
	go func() {
		<-sigs
		logger.DebugContext(ctx, "received SIGINT or SIGTERM")
		cancel()
		logger.InfoContext(ctx, "cancel generating fake data")
		os.Exit(1)
	}()
	logger.InfoContext(ctx, "fisnish generating fake data")
}
