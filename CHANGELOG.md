# Changelog

## 0.2.0

### Breaking changes

-   requires Go 1.21+ (per Go version policy).

### API changes

-   added `getTileFormatAndSize()` to attempt to detect the tile size in addition
    to the tile format; will read up to full image of first tile to detect tile
    size.
-   added `tilesize` to `MBtiles` struct.

### Bug fixes

-   fixed segfaults resulting from opening invalid MBTiles files (#6)
