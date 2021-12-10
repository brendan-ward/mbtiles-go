# MBTiles Reader for Go

A simple Go-based mbtiles reader.

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
