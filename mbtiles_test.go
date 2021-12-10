package mbtiles

import (
	"strings"
	"testing"
)

func Test_FindMBtiles(t *testing.T) {
	// there may be more test datasets, but we only check these ones
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
				found += 1
			}
		}
	}
	if found != len(expected) {
		t.Error("Did not list all expected mbtiles files in testdata directory")
	}
}

func Test_FindMBtiles_empty_dir(t *testing.T) {
	// empty directory should return no tilesets
	filenames, err := FindMBtiles("./examples")
	if err != nil {
		t.Error("Failed to list mbtiles in valid directory")
	}
	if len(filenames) != 0 {
		t.Error("Directory with no mbtiles returned non-empty result")
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
		path   string
		format TileFormat
	}{
		{path: "geography-class-jpg.mbtiles", format: JPG},
		{path: "geography-class-png.mbtiles", format: PNG},
		{path: "geography-class-webp.mbtiles", format: WEBP},
		{path: "world_cities.mbtiles", format: PBF},
	}

	for _, tc := range tests {
		db, err := Open("./testdata/" + tc.path)
		if err != nil {
			t.Error("Could not open:", tc.path)
			continue
		}
		if db.GetTileFormat() != tc.format {
			t.Error("Tile format does not match expected value for:", tc.path)
			continue
		}
	}
}

func Test_OpenMBtiles_invalid(t *testing.T) {
	tests := []struct {
		path string
		err  string
	}{
		{path: "invalid.mbtiles", err: "Missing one or more required tables: tiles, metadata"},
		{path: "invalid-tile-format.mbtiles", err: "Could not detect tile format"},
		{path: "incomplete.mbtiles", err: "Refusing to open mbtiles file with associated -journal file"},
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
