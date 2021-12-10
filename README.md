# MBTiles Reader for Go

A simple Go-based mbtiles reader.

![Build Status](https://github.com/brendan-ward/mbtiles-go/actions/workflows/test.yml/badge.svg)
[![Coverage Status](https://coveralls.io/repos/github/brendan-ward/mbtiles-go/badge.svg?branch=main)](https://coveralls.io/github/brendan-ward/mbtiles-go?branch=main)
[![GoDoc](https://godoc.org/github.com/brendan-ward/mbtiles-go?status.svg)](http://godoc.org/github.com/brendan-ward/mbtiles-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/brendan-ward/mbtiles-go)](https://goreportcard.com/report/github.com/brendan-ward/mbtiles-go)

Supports JPG, PNG, WebP, and vector tile tilesets created according to the
[mbtiles specification](https://github.com/mapbox/mbtiles-spec).

## Example:

```go
// Open an mbtiles file
db, err := mbtiles.Open("testdata/geography-class-jpg.mbtiles")
if err != nil { ... }
defer db.Close()

// read a tile into a byte slice
var data []byte
err = db.ReadTile(0, 0, 0, &data)
if err != nil { ... }
```

## Credits:

This was adapted from the `mbtiles` package in [mbtileserver](https://github.com/consbio/mbtileserver) to use the `crawshaw.io/sqlite` SQLite library.
