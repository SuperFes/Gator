module github.com/SuperFes/gator

go 1.23.4

replace internal/config => ./internal/config
replace database => ./database

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/lib/pq v1.10.9 // indirect
	internal/config v0.0.0-00010101000000-000000000000 // indirect
)
