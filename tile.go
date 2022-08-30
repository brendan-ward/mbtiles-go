package mbtiles

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image/jpeg"
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
	case GZIP:
		return "gzip"
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
func detectTileFormat(data []byte) (TileFormat, error) {
	for format, pattern := range formatPrefixes {
		if bytes.HasPrefix(data, pattern) {
			return format, nil
		}
	}

	return UNKNOWN, errors.New("could not detect tile format")
}

// detectTileSize reads tile dimensions from image tiles, and otherwise assumes
// 512px size for PBF tiles.  Tiles are assumed to be square.
// Data must contain at least the first 20 bytes of the beginning of a tile.
func detectTileSize(format TileFormat, data []byte) (uint32, error) {
	switch format {
	// PBF files are always 512px
	// GZIP masks PBF, which is only expected type for tiles in GZIP format
	case GZIP:
		return 512, nil
	case PBF:
		return 512, nil
	case PNG:
		// read the width from the IHDR chunk of the PNG
		if len(data) < 20 {
			return 0, errors.New("insufficient length to detect png image size")
		}
		return binary.BigEndian.Uint32(data[16:20]), nil
	case JPG:
		// JPG is a more complex structure, use the builtin JPG decoder
		cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			return 0, err
		}
		return uint32(cfg.Width), nil
	case WEBP:
		// Webp is a more complex structure with different bit-level encodings
		encType := data[12:16]
		switch {
		case bytes.HasPrefix(encType, []byte("VP8 ")): // Lossy
			// width appears to be at index 26-27
			if len(data) < 27 {
				return 0, errors.New("insufficient length to detect webp image size")
			}

			return uint32(int(data[27]&0x3f)<<8 | int(data[26])), nil

		case bytes.HasPrefix(encType, []byte("VP8L")): // Lossless
			// width is in 14 bits out of bytes 21-22
			if len(data) < 23 {
				return 0, errors.New("insufficient length to detect webp image size")
			}

			return uint32(binary.LittleEndian.Uint16(data[21:23])&0x1ff) + 1, nil

		case bytes.HasPrefix(encType, []byte("VP8X")): // Alpha
			// width is in 24 bits out of bytes 24-26
			if len(data) < 26 {
				return 0, errors.New("insufficient length to detect webp image size")
			}

			return uint32(binary.LittleEndian.Uint16(data[24:27])) + 1, nil
		}
	}

	return 0, nil
}
