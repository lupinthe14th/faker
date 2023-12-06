package main

import (
	"encoding/csv"
	"log/slog"
	"os"
	"sync"

	"github.com/brianvoe/gofakeit/v6"
)

const (
	batchSize  = 10000
	numWorkers = 10
	numRecords = 100000000
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	faker := gofakeit.NewCrypto()
	gofakeit.SetGlobalFaker(faker)

	file, err := os.Create("persons.csv")
	if err != nil {
		logger.Error("Error creating file: %v", err)
		os.Exit(1)
	}

	writer := csv.NewWriter(file)
	defer writer.Flush()

	personsChan := make(chan []string, batchSize)
	var wg sync.WaitGroup

	// ワーカーの起動
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numRecords/numWorkers; j++ {
				person := Person{}
				if err := gofakeit.Struct(&person); err != nil {
					logger.Error("Error creating person: %v", err)
					continue
				}

				personsChan <- []string{
					person.Name,
					person.Address,
					person.Phone,
					person.Country,
					person.Emoji,
					person.Movie.Genre,
				}
			}
		}()
	}

	var fileWg sync.WaitGroup
	fileWg.Add(1) // ファイル書き込み用のゴルーチンの終了を待つ

	// バッチ処理とCSVファイルへの書き込み
	go func() {
		defer fileWg.Done() // ファイル書き込み用のゴルーチンが終了したことを通知

		personsBatch := make([][]string, 0, batchSize)
		for person := range personsChan {
			personsBatch = append(personsBatch, person)
			if len(personsBatch) == batchSize {
				if err := writer.WriteAll(personsBatch); err != nil {
					logger.Error("Error writing to file: %v", err)
				}
				writer.Flush()
				personsBatch = make([][]string, 0, batchSize)
			}
		}
		if len(personsBatch) > 0 {
			if err := writer.WriteAll(personsBatch); err != nil {
				logger.Error("Error writing to file: %v", err)
			}
		}
	}()

	wg.Wait()
	close(personsChan)
	fileWg.Wait() // ファイル書き込み用のゴルーチンが終了するまで待つ
	writer.Flush()
	file.Close() // 全てのゴルーチンが終了した後にファイルを閉じる

}
