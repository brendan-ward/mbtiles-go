package mbtiles

import (
	"os"
	"strings"
	"testing"
	"time"
)

func Test_FindMBtiles(t *testing.T) {
	var expected = []string{
		"testdata/geography-class-jpg.mbtiles",
		"testdata/geography-class-png.mbtiles",
		"testdata/geography-class-webp.mbtiles",
		"testdata/world_cities.mbtiles",
	}

	filenames, err := FindMBtiles("./testdata")
	if err != nil {
		t.Error("Could not list mbtiles files in testdata directory")
	}

	found := 0

	for _, expectedFilename := range expected {
		for _, filename := range filenames {
			if filename == expectedFilename {
				found++
			}
		}
	}
	if found != len(expected) {
		t.Error("Did not list all expected mbtiles files in testdata directory")
	}
}

func Test_FindMBtiles_invalid_dir(t *testing.T) {
	// non-existing directory should fail with an error
	_, err := FindMBtiles("./invalid")
	if err == nil {
		t.Error("Did not fail to list mbtiles in invalid directory")
	}
}

func Test_OpenMBtiles(t *testing.T) {
	tests := []struct {
		path     string
		format   TileFormat
		tilesize uint32
	}{
		{path: "geography-class-jpg.mbtiles", format: JPG, tilesize: 256},
		{path: "geography-class-png.mbtiles", format: PNG, tilesize: 256},
		{path: "geography-class-webp.mbtiles", format: WEBP, tilesize: 256},
		{path: "world_cities.mbtiles", format: PBF, tilesize: 512},
	}

	for _, tc := range tests {
		db, err := Open("./testdata/" + tc.path)
		if err != nil {
			t.Error("Could not open:", tc.path)
			continue
		}

		if db.GetTileFormat() != tc.format {
			t.Error("Tile format", db.GetTileFormat(), "does not match expected value", tc.format, "for:", tc.path)
			continue
		}

		if db.GetTileSize() != tc.tilesize {
			t.Error("Tile size", db.GetTileSize(), "does not match expected value", tc.tilesize, "for:", tc.path)
			continue
		}
	}
}

func Test_OpenMBtiles_invalid(t *testing.T) {
	tests := []struct {
		path string
		err  string
	}{
		{path: "invalid.mbtiles", err: "missing one or more required tables: tiles, metadata"},
		{path: "invalid-tile-format.mbtiles", err: "could not detect tile format"},
		{path: "incomplete.mbtiles", err: "refusing to open mbtiles file with associated -journal file"},
		{path: "does-not-exist.mbtiles", err: "path does not exist"},
	}
	for _, tc := range tests {
		db, err := Open("./testdata/" + tc.path)
		if err == nil {
			t.Error("Invalid mbtiles did not raise error on open:", tc.path)
			continue
		}
		if db != nil {
			t.Error("Invalid mbtiles returned open handle:", tc.path)
		}
		if !strings.Contains(err.Error(), tc.err) {
			t.Error("Invalid mbtiles did not raise expected error:", tc.path, ", instead raised: ", err)
		}
	}
}

func Test_CloseMBtiles(t *testing.T) {
	// an MBtiles handle should not panic on close
	fakeDB := &MBtiles{}
	fakeDB.Close()
}

func Test_ReadMetadata(t *testing.T) {
	tests := []struct {
		path    string
		maxzoom int
	}{
		{path: "geography-class-jpg.mbtiles", maxzoom: 1},
		{path: "geography-class-png.mbtiles", maxzoom: 1},
		{path: "geography-class-png-missing-metadata.mbtiles", maxzoom: 1},
		{path: "geography-class-webp.mbtiles", maxzoom: 1},
		{path: "world_cities.mbtiles", maxzoom: 6},
	}

	requiredKeys := []string{
		"minzoom", "maxzoom", "name",
	}

	for _, tc := range tests {
		db, err := Open("./testdata/" + tc.path)
		if err != nil {
			t.Error("Could not open:", tc.path)
			continue
		}
		metadata, err := db.ReadMetadata()

		if err != nil {
			t.Error("Could not read metadata for:", tc.path)
			continue
		}
		if metadata == nil {
			t.Error("ReadMetadata returned empty results for:", tc.path)
			continue
		}

		for _, key := range requiredKeys {
			_, ok := metadata[key]
			if !ok {
				t.Error("Missing required metadata key", key, " for: ", tc.path)
				break
			}
		}

		if metadata["maxzoom"] != tc.maxzoom {
			t.Error("maxzoom is not expected value for:", tc.path, "got", metadata["maxzoom"])
		}
	}
}

func Test_ReadMetadata_contents(t *testing.T) {
	db, _ := Open("./testdata/geography-class-png.mbtiles")

	expectedMetadata := map[string]interface{}{
		"name":        "Geography Class",
		"description": "One of the example maps that comes with TileMill - a bright & colorful world map that blends retro and high-tech with its folded paper texture and interactive flag tooltips. ",
		"minzoom":     0,
		"maxzoom":     1,
	}
	metadata, err := db.ReadMetadata()
	if err != nil {
		t.Error("Error raised when reading metadata")
	}
	for key, expectedValue := range expectedMetadata {
		value, ok := metadata[key]
		if !ok {
			t.Errorf("Metadata missing expected key: %q", key)
		}
		if value != expectedValue {
			t.Errorf("Metadata value '%v' does not match expected value '%v'", value, expectedValue)
		}
	}
	var expectedBounds = []float64{-180, -85.0511, 180, 85.0511}
	bounds, ok := metadata["bounds"]
	if !ok {
		t.Error("Metadata missing expected key: bounds")
	}
	boundsValues := bounds.([]float64)
	if len(boundsValues) != 4 {
		t.Error("Metadata bounds not expected length")
	}
	for i, expectedValue := range expectedBounds {
		if boundsValues[i] != expectedValue {
			t.Errorf("Metadata bounds does not have expected values.  Found: %v expected: %v", boundsValues[i], expectedValue)
		}
	}
}

func Test_ReadTile(t *testing.T) {
	tests := []struct {
		z     int64
		x     int64
		y     int64
		bytes int
	}{
		{z: 0, x: 0, y: 0, bytes: 21246},
		{z: 1, x: 0, y: 0, bytes: 13843},
		// notexistant tile, returns 0 bytes
		{z: 10, x: 0, y: 0, bytes: 0},
	}

	db, _ := Open("./testdata/geography-class-png.mbtiles")

	for _, tc := range tests {
		var data []byte
		err := db.ReadTile(tc.z, tc.x, tc.y, &data)
		if err != nil {
			t.Error("Unexpected error reading tile:", tc.z, tc.x, tc.y)
			continue
		}
		if len(data) != tc.bytes {
			t.Error("ReadTile returned different number of bytes than expected for tile:", tc.z, tc.x, tc.y, "got:", len(data))
			continue
		}
	}
}

func Test_GetFilename(t *testing.T) {
	filename := "./testdata/geography-class-png.mbtiles"
	db, _ := Open(filename)
	defer db.Close()

	if db.GetFilename() != filename {
		t.Error("GetFilename does not match expected value, got:", db.GetFilename())
	}
}

func Test_GetTimestamp(t *testing.T) {
	filename := "./testdata/geography-class-png.mbtiles"
	stat, _ := os.Stat(filename)
	expected := stat.ModTime().Round(time.Second)

	db, _ := Open(filename)
	defer db.Close()

	if db.GetTimestamp() != expected {
		t.Error("Timestamp does not match value from os.Stat, got:", db.GetTimestamp())
	}
}
