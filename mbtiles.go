package mbtiles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
)

// MBtiles provides a basic handle for an mbtiles file.
type MBtiles struct {
	filename  string
	pool      *sqlitex.Pool
	format    TileFormat
	timestamp time.Time
	tilesize  uint32
}

// FindMBtiles recursively finds all mbtiles files within a given path.
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

// OpenInMemory opens an MBtiles file for reading, and validates that it has the correct
// structure. Then it loads it to in-memory database. Use this function only with files small enough to be
// loaded in-memory.
func OpenInMemory(ctx context.Context, path string) (*MBtiles, error) {
	modTime, err := fileModTime(path)
	if err != nil {
		return nil, err
	}

	format, tilesize, err := validateAndGetFormatAndSize(path)
	if err != nil {
		return nil, err
	}

	srcCon, err := sqlite.OpenConn(path, sqlite.SQLITE_OPEN_READONLY)
	if err != nil {
		return nil, err
	}
	defer srcCon.Close()

	inMemoryPath := "file::memory:?mode=memory"
	dstCon, err := sqlite.OpenConn(inMemoryPath, sqlite.SQLITE_OPEN_CREATE|sqlite.SQLITE_OPEN_READWRITE|sqlite.SQLITE_OPEN_URI)
	if err != nil {
		return nil, err
	}
	defer dstCon.Close()

	bkp, err := srcCon.BackupInit("", "", dstCon)
	if err != nil {
		return nil, fmt.Errorf("backup %s to in memory db: %w", path, err)
	}
	defer bkp.Finish()

	if err := bkp.Step(-1); err != nil {
		return nil, fmt.Errorf("transfer whole db: %w", err)
	}

	pool, err := sqlitex.Open(inMemoryPath, sqlite.SQLITE_OPEN_READONLY|sqlite.SQLITE_OPEN_URI|sqlite.SQLITE_OPEN_NOMUTEX, 10)
	if err != nil {
		return nil, err
	}

	return &MBtiles{
		filename:  inMemoryPath,
		pool:      pool,
		timestamp: modTime,
		format:    format,
		tilesize:  tilesize,
	}, nil
}

// Open opens an MBtiles file for reading, and validates that it has the correct
// structure.
func Open(path string) (*MBtiles, error) {
	modTime, err := fileModTime(path)
	if err != nil {
		return nil, err
	}

	format, tilesize, err := validateAndGetFormatAndSize(path)
	if err != nil {
		return nil, err
	}

	pool, err := sqlitex.Open(path, sqlite.SQLITE_OPEN_READONLY|sqlite.SQLITE_OPEN_NOMUTEX, 10)
	if err != nil {
		return nil, err
	}

	db := &MBtiles{
		filename:  path,
		pool:      pool,
		timestamp: modTime,
		format:    format,
		tilesize:  tilesize,
	}

	return db, nil
}

func fileModTime(path string) (time.Time, error) {
	stat, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return time.Time{}, fmt.Errorf("path does not exist: %q", path)
		}
		return time.Time{}, err
	}
	return stat.ModTime().Round(time.Second), nil
}

func validateAndGetFormatAndSize(path string) (TileFormat, uint32, error) {
	// there must not be a corresponding *-journal file (tileset is still being created)
	if _, err := os.Stat(path + "-journal"); err == nil {
		return 0, 0, fmt.Errorf("refusing to open mbtiles file with associated -journal file (incomplete tileset)")
	}

	// open a single connection first while we are verifying the database
	// since there are issues closing out a connection pool on error here
	con, err := sqlite.OpenConn(path, sqlite.SQLITE_OPEN_READONLY|sqlite.SQLITE_OPEN_NOMUTEX)
	if con != nil {
		defer con.Close()
	}
	if err != nil {
		return 0, 0, err
	}

	err = validateRequiredTables(con)
	if err != nil {
		return 0, 0, err
	}
	return getTileFormatAndSize(con)
}

// Close closes a MBtiles file
func (db *MBtiles) Close() {
	if db.pool != nil {
		db.pool.Close()
	}
}

// ReadTile reads a tile for z, x, y into the provided *[]byte.
// data will be nil if the tile does not exist in the database
func (db *MBtiles) ReadTile(z int64, x int64, y int64, data *[]byte) error {
	if db == nil || db.pool == nil {
		return errors.New("cannot read tile from closed mbtiles database")
	}

	con, err := db.getConnection(context.TODO())
	defer db.closeConnection(con)
	if err != nil {
		return err
	}

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

// ReadMetadata reads the metadata table into a map, casting their values into
// the appropriate type
func (db *MBtiles) ReadMetadata() (map[string]interface{}, error) {
	if db == nil || db.pool == nil {
		return nil, errors.New("cannot read tile from closed mbtiles database")
	}

	con, err := db.getConnection(context.TODO())
	defer db.closeConnection(con)
	if err != nil {
		return nil, err
	}

	var (
		key   string
		value string
	)
	metadata := make(map[string]interface{})

	query, err := con.Prepare("select name, value from metadata where value is not ''")
	if err != nil {
		return nil, err
	}
	defer query.Reset()

	for {
		hasRow, err := query.Step()
		if err != nil {
			return nil, err
		}
		if !hasRow {
			break
		}

		key = query.GetText("name")
		value = query.GetText("value")

		switch key {
		case "maxzoom", "minzoom":
			metadata[key], err = strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("cannot read metadata item %s: %v", key, err)
			}
		case "bounds", "center":
			metadata[key], err = parseFloats(value)
			if err != nil {
				return nil, fmt.Errorf("cannot read metadata item %s: %v", key, err)
			}
		case "json":
			err = json.Unmarshal([]byte(value), &metadata)
			if err != nil {
				return nil, fmt.Errorf("unable to parse JSON metadata item: %v", err)
			}
		default:
			metadata[key] = value
		}
	}

	// Supplement missing values by inferring from available data
	_, hasMinZoom := metadata["minzoom"]
	_, hasMaxZoom := metadata["maxzoom"]
	if !(hasMinZoom && hasMaxZoom) {
		q2, err := con.Prepare("select min(zoom_level), max(zoom_level) from tiles")
		if err != nil {
			return nil, err
		}
		defer q2.Reset()
		_, err = q2.Step()
		if err != nil {
			return nil, err
		}

		metadata["minzoom"] = q2.ColumnInt(0)
		metadata["maxzoom"] = q2.ColumnInt(1)
	}
	return metadata, nil
}

func (db *MBtiles) GetFilename() string {
	return db.filename
}

// GetTileFormat returns the TileFormat of the mbtiles file.
func (db *MBtiles) GetTileFormat() TileFormat {
	return db.format
}

// GetTileSize returns the tile size in pixels of the mbtiles file, if detected.
// Returns 0 if tile size is not detected.
func (db *MBtiles) GetTileSize() uint32 {
	return db.tilesize
}

// Timestamp returns the time stamp of the mbtiles file.
func (db *MBtiles) GetTimestamp() time.Time {
	return db.timestamp
}

// getConnection gets a sqlite.Conn from an open connection pool.
// closeConnection(con) must be called to release the connection.
func (db *MBtiles) getConnection(ctx context.Context) (*sqlite.Conn, error) {
	con := db.pool.Get(ctx)
	if con == nil {
		return nil, errors.New("connection could not be opened")
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
	if err != nil {
		return err
	}
	defer query.Finalize()

	_, err = query.Step()
	if err != nil {
		return err
	}

	if query.ColumnInt32(0) < 2 {
		return errors.New("missing one or more required tables: tiles, metadata")
	}
	return nil
}

// getTileFormat reads the first 8 bytes of the first tile in the database.
// See TileFormat for list of supported tile formats.
func getTileFormat(con *sqlite.Conn) (TileFormat, error) {
	query, _, err := con.PrepareTransient("select tile_data from tiles limit 1")
	if err != nil {
		return UNKNOWN, err
	}
	defer query.Finalize()

	hasRow, err := query.Step()
	if err != nil {
		return UNKNOWN, err
	}
	if !hasRow {
		return UNKNOWN, errors.New("'tiles' table must be non-empty")
	}

	r := query.ColumnReader(0)
	if r.Size() < 8 {
		return UNKNOWN, errors.New("tile data too small to determine tile format")
	}

	magicWord := make([]byte, 8)
	_, err = r.Read(magicWord)
	if err != nil {
		return UNKNOWN, err
	}

	format, err := detectTileFormat(magicWord)
	if err != nil {
		return UNKNOWN, err
	}

	// GZIP masks PBF, which is only expected type for tiles in GZIP format
	if format == GZIP {
		format = PBF
	}

	return format, nil
}

// getTileFormatAndSize reads the first tile in the database to detect the tile
// format and if PNG also the size.
// See TileFormat for list of supported tile formats.
func getTileFormatAndSize(con *sqlite.Conn) (TileFormat, uint32, error) {
	var tilesize uint32 = 0 // not detected for all formats

	query, _, err := con.PrepareTransient("select tile_data from tiles limit 1")
	if err != nil {
		return UNKNOWN, tilesize, err
	}
	defer query.Finalize()

	hasRow, err := query.Step()
	if err != nil {
		return UNKNOWN, tilesize, err
	}
	if !hasRow {
		return UNKNOWN, tilesize, errors.New("'tiles' table must be non-empty")
	}

	var tileData = make([]byte, query.ColumnLen(0))
	query.ColumnBytes(0, tileData)

	format, err := detectTileFormat(tileData)
	if err != nil {
		return UNKNOWN, tilesize, err
	}

	// GZIP masks PBF, which is only expected type for tiles in GZIP format
	if format == GZIP {
		format = PBF
	}

	tilesize, err = detectTileSize(format, tileData)
	if err != nil {
		return format, tilesize, err
	}

	return format, tilesize, nil
}

// parseFloats converts a commma-delimited string of floats to a slice of
// float64 and returns it and the first error that was encountered.
// Example: "1.5,2.1" => [1.5, 2.1]
func parseFloats(str string) ([]float64, error) {
	split := strings.Split(str, ",")
	var out []float64
	for _, v := range split {
		value, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return out, fmt.Errorf("could not parse %q to floats: %v", str, err)
		}
		out = append(out, value)
	}
	return out, nil
}
