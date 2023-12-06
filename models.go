package main

import "github.com/brianvoe/gofakeit/v6"

type Person struct {
	Name    string              `fake:"{firstname} {lastname}"`
	Address string              `fake:"{address}"`
	Phone   string              `fake:"{phoneformatted}"`
	Country string              `fake:"{country}"`
	Emoji   string              `fake:"{emoji}"` // New in v6.0.0
	Movie   *gofakeit.MovieInfo `fake:"{movie}"`
}
