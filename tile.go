package mbtiles

import (
	"bytes"
	"errors"
)

// TileFormat defines the tile format of tiles an mbtiles file.  Supported image
// formats:
//   * PNG
//   * JPG
//   * WEBP
//   * PBF  (vector tile protocol buffers)
// Tiles may be compressed, in which case the type is one of:
//   * GZIP (assumed to be GZIP'd PBF data)
//   * ZLIB
type TileFormat uint8

// TileFormat enum values
const (
	UNKNOWN TileFormat = iota // UNKNOWN TileFormat cannot be determined from first few bytes of tile
	GZIP                      // encoding = gzip
	ZLIB                      // encoding = deflate
	PNG
	JPG
	PBF
	WEBP
)

// String returns a string representing the TileFormat.
func (t TileFormat) String() string {
	switch t {
	case PNG:
		return "png"
	case JPG:
		return "jpg"
	case PBF:
		return "pbf"
	case WEBP:
		return "webp"
	default:
		return ""
	}
}

// MimeType returns the MIME content type for the TileFormat
func (t TileFormat) MimeType() string {
	switch t {
	case PNG:
		return "image/png"
	case JPG:
		return "image/jpeg"
	case PBF:
		return "application/x-protobuf" // Content-Encoding header must be gzip
	case WEBP:
		return "image/webp"
	default:
		return ""
	}
}

var formatPrefixes = map[TileFormat][]byte{
	GZIP: []byte("\x1f\x8b"), // this masks PBF format too
	ZLIB: []byte("\x78\x9c"),
	PNG:  []byte("\x89\x50\x4E\x47\x0D\x0A\x1A\x0A"),
	JPG:  []byte("\xFF\xD8\xFF"),
	// NOTE: this is technically only the RIFF part of the header,
	// but none of the other RIFF file formats are likely to be stored
	// as tiles.
	WEBP: []byte("\x52\x49\x46\x46"),
}

// detectFileFormat inspects the first few bytes of byte array to determine tile
// format PBF tile format does not have a distinct signature, it will be
// returned as GZIP, and it is up to caller to determine that it is a PBF format.
func detectTileFormat(data *[]byte) (TileFormat, error) {
	for format, pattern := range formatPrefixes {
		if bytes.HasPrefix(*data, pattern) {
			return format, nil
		}
	}

	return UNKNOWN, errors.New("could not detect tile format")
}
