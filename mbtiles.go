package mbtiles

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
)

// MBtiles provides a basic handle for an mbtiles file.
type MBtiles struct {
	pool      *sqlitex.Pool
	format    TileFormat
	timestamp time.Time
}

// FindMBTiles recursively finds all mbtiles files within a given path.
func FindMBtiles(path string) ([]string, error) {
	var filenames []string
	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Ignore any that have an associated -journal file; these are incomplete
		if _, err := os.Stat(p + "-journal"); err == nil {
			return nil
		}
		if ext := filepath.Ext(p); ext == ".mbtiles" {
			filenames = append(filenames, p)

		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return filenames, err
}

// Open opens an MBtiles file for reading, and validates that it has the correct
// structure.
func Open(path string) (*MBtiles, error) {
	// try to open file; fail fast if it doesn't exist
	stat, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("Path does not exist: %q", path)
		}
		return nil, err
	}

	// there must not be a corresponding *-journal file (tileset is still being created)
	if _, err := os.Stat(path + "-journal"); err == nil {
		return nil, fmt.Errorf("Refusing to open mbtiles file with associated -journal file (incomplete tileset)")
	}

	pool, err := sqlitex.Open(path, sqlite.SQLITE_OPEN_READONLY|sqlite.SQLITE_OPEN_NOMUTEX, 10)
	if err != nil {
		return nil, err
	}

	db := &MBtiles{
		pool:      pool,
		timestamp: stat.ModTime().Round(time.Second),
	}

	con, err := db.getConnection(nil)
	if err != nil {
		return nil, err
	}
	defer db.closeConnection(con)

	err = validateRequiredTables(con)
	if err != nil {
		return nil, err
	}

	format, err := getTileFormat(con)
	if err != nil {
		return nil, err
	}

	db.format = format

	return db, nil
}

// Close closes a MBtiles file
func (db *MBtiles) Close() {
	if db != nil && db.pool != nil {
		db.pool.Close()
	}
}

// ReadTile reads a tile for z, x, y into the provided *[]byte.
// data will be nil if the tile does not exist in the database
func (db *MBtiles) ReadTile(z int64, x int64, y int64, data *[]byte) error {
	if db == nil || db.pool == nil {
		return errors.New("Cannot read tile from closed mbtiles database")
	}

	con, err := db.getConnection(nil)
	if err != nil {
		return err
	}
	defer db.closeConnection(con)

	query, err := con.Prepare("select tile_data from tiles where zoom_level = $z and tile_column = $x and tile_row = $y")
	if err != nil {
		return err
	}
	defer query.Reset()

	query.SetInt64("$z", z)
	query.SetInt64("$x", x)
	query.SetInt64("$y", y)

	hasRow, err := query.Step()
	if err != nil {
		return err
	}

	// If this tile does not exist in the database, return empty bytes
	if !hasRow {
		*data = nil
		return nil
	}

	var tileData = make([]byte, query.ColumnLen(0))
	query.ColumnBytes(0, tileData)
	*data = tileData[:]

	if err != nil {
		return err
	}

	return nil
}

// GetTileFormat returns the TileFormat of the mbtiles file.
func (db *MBtiles) GetTileFormat() TileFormat {
	return db.format
}

// TimeStamp returns the time stamp of the mbtiles file.
func (db *MBtiles) TimeStamp() time.Time {
	return db.timestamp
}

// getConnection gets a sqlite.Conn from an open connection pool.
// closeConnection(con) must be called to release the connection.
func (db *MBtiles) getConnection(ctx context.Context) (*sqlite.Conn, error) {
	con := db.pool.Get(ctx)
	if con == nil {
		return nil, errors.New("Connection could not be opened")
	}
	return con, nil
}

// closeConnection closes an open sqlite.Conn and returns it to the pool.
func (db *MBtiles) closeConnection(con *sqlite.Conn) {
	if con != nil {
		db.pool.Put(con)
	}
}

// validateRequiredTables checks that both 'tiles' and 'metadata' tables are
// present in the database
func validateRequiredTables(con *sqlite.Conn) error {
	query, _, err := con.PrepareTransient("SELECT count(*) as c FROM sqlite_master WHERE name in ('tiles', 'metadata')")
	defer query.Finalize()

	if err != nil {
		return err
	}

	_, err = query.Step()
	if err != nil {
		return err
	}

	if query.ColumnInt32(0) < 2 {
		return errors.New("Missing one or more required tables: tiles, metadata")
	}
	return nil
}

// getTileFormat reads the first 8 bytes of the first tile in the database.
// See TileFormat for list of supported tile formats.
func getTileFormat(con *sqlite.Conn) (TileFormat, error) {
	query, _, err := con.PrepareTransient("select tile_data from tiles limit 1")
	defer query.Finalize()

	if err != nil {
		return UNKNOWN, err
	}

	hasRow, err := query.Step()
	if err != nil {
		return UNKNOWN, err
	}
	if !hasRow {
		return UNKNOWN, errors.New("'tiles' table must be non-empty")
	}

	r := query.ColumnReader(0)
	if r.Size() < 8 {
		return UNKNOWN, errors.New("Tile data too small to determine tile format")
	}

	magicWord := make([]byte, 8)
	_, err = r.Read(magicWord)
	if err != nil {
		return UNKNOWN, err
	}

	format, err := detectTileFormat(&magicWord)
	if err != nil {
		return UNKNOWN, err
	}

	// GZIP masks PBF, which is only expected type for tiles in GZIP format
	if format == GZIP {
		format = PBF
	}

	return format, nil
}
