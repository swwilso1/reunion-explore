# reunion-explore

A CLI and web server for parsing and exploring [Reunion 14](https://www.leisterpro.com) genealogy bundles (`.familyfile14`). Reunion 10 (`.familyfile10`) and Reunion 13 (`.familyfile13`) bundles are also supported, and any of them can be exported to GEDCOM.

## Install

Download a prebuilt binary from [Releases](https://github.com/kedoco/reunion-explore/releases), or build from source:

```sh
go install github.com/kedoco/reunion-explore/cmd/reunion-explore@latest
```

## Usage

```
reunion-explore <command> <bundle>
```

All commands accept a `-j` / `--json` flag for JSON output.

### Commands

| Command | Description |
|---------|-------------|
| `json <bundle>` | Dump full family file as JSON |
| `stats <bundle>` | Summary counts (persons, families, places, etc.) |
| `persons <bundle>` | List all persons (`--surname` to filter) |
| `person <bundle> <id>` | Detail view for a person |
| `search <bundle> <query>` | Search person names |
| `couples <bundle>` | List all couples |
| `ancestors <bundle> <id>` | Walk ancestor tree (`-g` for max generations) |
| `descendants <bundle> <id>` | Walk descendant tree (`-g` for max generations) |
| `treetops <bundle> <id>` | List terminal ancestors (no parents) |
| `summary <bundle> <id>` | Per-person stats (spouses, ancestors, surnames) |
| `places <bundle>` | List all places |
| `events <bundle>` | List all event type definitions |
| `gedcom <bundle>` | Export as GEDCOM 5.5.1 (`-o` for an output file, default stdout; `--no-pictures` to omit picture links) |
| `serve <bundle>` | Start web server (`-a` for listen address, default `:8080`) |

### Examples

```sh
# Show file statistics
reunion-explore stats ~/Documents/MyFamily.familyfile14

# Search for a person
reunion-explore search ~/Documents/MyFamily.familyfile14 "Smith"

# View ancestors up to 5 generations
reunion-explore ancestors ~/Documents/MyFamily.familyfile14 42 -g 5

# Export to GEDCOM
reunion-explore gedcom ~/Documents/MyFamily.familyfile14 -o MyFamily.ged

# Start the web UI
reunion-explore serve ~/Documents/MyFamily.familyfile14 -a :3000
```

### Web Server

The `serve` command starts an HTTP server with a REST API and embedded web UI.

API endpoints are available under `/api/` — see `/api/openapi.json` for the full OpenAPI 3.1.0 spec.

## Versioning

Release tags follow plain semver (`vX.Y.Z`). The supported Reunion format version is indicated in the `--version` output and release notes (e.g. "reunion14" means Reunion 14 compatibility).

## File Format

Reunion 14 uses a proprietary, undocumented binary format. Leister Productions has never published a specification — a developer [noted on ReunionTalk](https://www.reuniontalk.com/forum/using-older-versions-of-reunion/9509-conflicted-file-from-dropbox) that *"viewing the contents of a Family File package was not something we intended users to do."* Everything below was reverse-engineered from the Reunion 14 demo's sample file and confirmed against the parser in this repo.

A sample file ships with the Reunion 14 demo at `~/Documents/Reunion Files/Sample Family 14.familyfile14`.

### GEDCOM Export

`gedcom` writes GEDCOM 5.5.1 in UTF-8. Events use the GEDCOM tag from each Reunion event type (BIRT, DEAT, MARR, ...); event types without a standard tag are written as `EVEN`/`FACT` with a `TYPE`. Notes, sources (title, author, publisher, other fields as a note), source citations with page details, and picture file paths are included. Reunion-only data such as child-status colors and picture crop settings is not exported.

### Reunion 10

Reunion 10 bundles store the data file as `Family File.familyfile` and set the high bit (0x8000) on record type codes; otherwise records use the same format as later versions. Notes are stored as RTF (after an `ntsa` marker) and are converted to plain text, and pictures may be referenced by classic Mac alias records, from which the file path is extracted.

### Reunion 13

Reunion 13 bundles use the same layout and record format as Reunion 14; the familydata header carries format version `0x0d` instead of `0x0e` at offset 0x0C. The one structural difference handled by the parser is that Reunion 13 stores a placeholder record with ID `0xFFFFFFFF` for each record type, which is skipped. A Reunion 13 sample is included at `testdata/Sample Family 13.familyfile13`.

The only known external reference is the [Archive Team wiki page on Reunion](http://fileformats.archiveteam.org/wiki/Reunion), which lists magic bytes and version history but no detailed specification. No other parsers, libraries, or format documentation exist — this project is the only known implementation.

### Bundle Structure

A `.familyfile14` is a macOS [bundle](https://developer.apple.com/library/archive/documentation/CoreFoundation/Conceptual/CFBundles/BundleTypes/BundleTypes.html) (a directory that Finder displays as a single file). Right-click → "Show Package Contents" reveals:

```
MyFamily.familyfile14/
├── familyfile.familydata        # Binary data — all persons, families, places, events, sources, notes, media
├── familyfile.signature         # Text file containing a checksum (e.g. "1579320")
├── places.cache                 # Full-length place names (magic: "ahcp")
├── placeUsage.cache             # Place-to-event cross-references (magic: "hcup")
├── fmnames.cache                # Given/first names index (magic: "2wps")
├── surnames.cache               # Surname index (magic: "10ns")
├── shNames.cache                # Searchable full names (magic: "10hSan")
├── shGeneral.cache              # General search index (magic: "10hSeg")
├── timestamps.cache             # Timestamp records (magic: "icst")
├── globalRecords.cache          # Global record metadata (magic: "rblg")
├── bookmarks.cache              # User bookmarks (magic: "2kmb")
├── colortags.cache              # Color tag assignments (magic: "actc")
├── colortagsettings.cache       # Color tag display settings (magic: "gatc")
├── associations.cache           # Person associations (magic: "cosa")
├── noteboard.cache              # Noteboard entries (magic: "10bn")
├── relatives.cache              # Relative relationships (magic: "cler")
├── descriptions.cache           # File descriptions (magic: "idst")
└── thumbnails/
    ├── thumbnails_large/        # Large preview JPEGs (1000px)
    │   ├── p1-2d2f3-1000.jpg    # Person thumbnails: p{personID}-{hash}-{size}.jpg
    │   └── f1-dea36-1000.jpg    # Family thumbnails: f{familyID}-{hash}-{size}.jpg
    ├── thumbnails_small/        # Small preview JPEGs (200px)
    │   └── ...
    ├── thumbnails_src_large/    # Source record thumbnails (large)
    └── thumbnails_src_small/    # Source record thumbnails (small)
```

Backups created by Reunion exclude cache files (they are regenerated on first open), which is why backups are smaller than the working file.

### `familyfile.familydata` — Binary Format

This is the core data file containing all genealogical records.

#### File Header

The file begins with a fixed header:

```
Offset  Size  Description
──────  ────  ───────────
0x00    8     Magic string: "3SDUAU~R"
0x08    72    Reserved binary data
0x50    var   Device ID (newline-terminated ASCII string)
        var   Model name (newline-terminated, e.g. "Kevin\u2019s Mac mini")
        var   Serial number (newline-terminated)
        var   App path (null-terminated)
```

Older versions used different magic strings: Reunion 8 used `"UDS3R~U8"`, Reunion 9 used `"3SDU9U~R"`.

#### Record Scanning

After the header, all data is stored as a sequence of records. Records are located by scanning for a 4-byte **marker pattern**: `05 03 02 01`.

Each record has this layout:

```
Offset     Size  Description
──────     ────  ───────────
marker-8   4     Padding / zeros (may contain overflow data from previous record)
marker-4   2     Sequence number (uint16 LE)
marker-2   2     Record type code (uint16 LE)
marker+0   4     Marker: 05 03 02 01
marker+4   4     Data length (uint32 LE)
marker+8   4     Record ID (uint32 LE)
marker+12  var   Record data (length from marker+4)
```

#### Record Types

| Type Code | Name   | Description                      |
|-----------|--------|----------------------------------|
| `0x20C4`  | Person | Individual person record         |
| `0x20C8`  | Family | Family group (couple + children) |
| `0x20CC`  | Schema | Event type definition            |
| `0x20D0`  | Source | Source/citation record           |
| `0x20D4`  | Media  | Media reference record           |
| `0x20D8`  | Place  | Place record                     |
| `0x2104`  | Note   | Inline note                      |
| `0x2108`  | Doc    | Document record                  |
| `0x210C`  | Report | Report record                    |

### TLV Field Encoding

Person, Family, Schema, Source, and Media records encode their fields using a Tag-Length-Value (TLV) scheme. The record data starts with a 6-byte preamble, followed by TLV fields:

```
Record data:
  Offset 0-3:   4-byte timestamp
  Offset 4-5:   2-byte size (repeated)
  Offset 6+:    TLV fields

Each TLV field:
  Offset 0-1:   Total length (uint16 LE) — includes the 4-byte header itself
  Offset 2-3:   Tag (uint16 LE)
  Offset 4+:    Field data (total_length - 4 bytes)
```

A `total_length < 4` terminates parsing.

#### Person Field Tags (`0x20C4`)

| Tag      | Description          | Encoding                           |
|----------|----------------------|------------------------------------|
| `0x000C` | Surname (primary)    | Null-padded string                 |
| `0x001B` | Sex                  | 1 byte: `1`=male, `2`=female      |
| `0x001E` | Given name           | Null-padded string                 |
| `0x0020` | Name source citations| Source citation block (see below)  |
| `0x0023` | Surname (secondary)  | Null-padded string                 |
| `0x0028` | Prefix title         | Null-padded string (e.g. "Dr.")    |
| `0x002D` | Suffix title         | Null-padded string (e.g. "Jr.", "III") |
| `0x0037` | User ID              | Null-padded string                 |
| `≥0x0100`| Events               | Event sub-structure (see below)    |

Events with tags `< 0x03E8` may contain inline note references. Events with tags `0x03E8`–`0x0BB7` are life events (birth, death, etc.). Events with tags `≥ 0x0BB8` are facts (occupation, religion, etc.).

#### Family Field Tags (`0x20C8`)

| Tag          | Description  | Encoding                                |
|--------------|--------------|-----------------------------------------|
| `0x0050`     | Partner 1 ID | uint32 LE                               |
| `0x0051`     | Partner 2 ID | uint32 LE                               |
| `0x005F`     | Marriage     | Marriage event data                     |
| `0x00FA–00FF`| Children     | uint32 LE, actual child ID = `value >> 8` |
| `≥0x0100`    | Events       | Event sub-structure (see below)         |

#### Schema / Event Definition Field Tags (`0x20CC`)

| Tag      | Description    | Encoding |
|----------|----------------|----------|
| `0x0014` | Display name   | String   |
| `0x0019` | GEDCOM code    | String   |
| `0x0028` | Short label    | String   |
| `0x0032` | Abbreviation   | String   |
| `0x0037` | Abbreviation 2 | String   |
| `0x003C` | Abbreviation 3 | String   |
| `0x006E` | Sentence form  | String   |
| `0x0078` | Preposition    | String   |

### Event Sub-Structure

Event fields (tags `≥ 0x0100`) contain a nested structure:

```
Offset 0-1:    2-byte sub-header (event-specific metadata, not date-related)
Offset 2-3:    2-byte sub-header continuation
Offset 4-17:   14 bytes sub-header continuation (schema ID at offset 16 as uint16 LE)
Offset 18+:    Sub-TLV fields
```

Place references are embedded as `[[pt:NNN]]` text patterns within the raw event data.

#### Event Sub-TLV Fields

Sub-TLV fields at offset 18 within event data follow the standard TLV encoding (length u16 LE + tag u16 LE + data). All use tag `0x0000`. They appear in this order:

| Sub-TLV | Length | Content |
|---------|--------|---------|
| Date | Exactly 8 | 4-byte date encoding (see Date Encoding) |
| Place ref | Varies | `[[pt:NNN]]` text |
| Memo / Fact text | Varies, starts with printable byte | Free-text memo (events) or fact value (facts, e.g. "Senator") |
| Source citations | Varies, starts with binary header | Citation block: `innerLength(u32LE) + count(u32LE) + entries` |

The memo and citation sub-TLVs are distinguished by their first byte: memo text starts with a printable character (`≥ 0x20`), while citation data starts with a binary length field.

#### Source Citation Encoding

Source citations appear both as person-level fields (tag `0x0020`) and as the last sub-TLV within event data:

```
Offset  Size   Description
──────  ────   ───────────
0       4      Inner length (uint32 LE)
4       4      Citation count (uint32 LE)
Per entry:
  0     2      Entry length (uint16 LE, includes this field)
  2     2      Unknown / hash
  4     4      Source record ID (uint32 LE)
  8     N      Detail text (entry_length - 8 bytes), null-stripped
```

#### Date Encoding

Dates are stored in a sub-TLV field at offset 18 with total length = 8:

```
Byte 22:       Precision flags
                 0x00 = exact
                 0xA0 = approximate ("about"), year-only
                 0x40 = "after"
                 0xE0 = "after", year-only

Byte 23:       Day + month offset
                 Bits 7-6: month offset within group (0-3)
                 Bits 5-0: day of month (0 = unknown)

Bytes 24-25:   Year + month group (uint16 LE)
                 totalQ = (year + 8000) * 4 + group
                 group: 0 → months 1-3, 1 → months 4-7, 2 → months 8-11, 3 → month 12
```

The month is split across two bytes: `month = group * 4 + offset`, yielding 1–12. For example, November (month 11) encodes as group=2, offset=3 — so `totalQ` ends in binary `10` and the day byte has bits 7-6 = `11`.

### Note Record Format (`0x2104`)

Note records use a different layout from TLV-based records. After the record header (ID, sequence number, data length), the record data contains:

```
Offset 0-7:    8-byte binary preamble
Offset 8+:     Binary header followed by text content
```

The text content starts at the first `«` character (U+00AB, bytes `0xC2 0xAB`). Notes with formatting use markup delimiters `«tag»...«/tag»` (see Markup below). Plain-text notes (no formatting) contain raw text without markup tags.

After the text, records are null-padded and end with a small binary footer (typically `NN 00 04 21`, where `04 21` is the note record type `0x2104` in little-endian). The parser trims this trailer by:
1. Finding the last closing markup tag `«/...»` and trimming after it, or
2. For plain-text notes, trimming at the first null byte.

#### Note Markup

Formatted notes use `«»` (guillemet) delimiters for inline markup:

| Tag | Meaning |
|-----|---------|
| `«ff=1»...«/ff»` | Font/format wrapper (top-level) |
| `«b»...«/b»` | Bold |
| `«i»...«/i»` | Italic |
| `«u»...«/u»` | Underline |
| `«c=RRGGBBAA»...«/c»` | Text color (hex RGBA) |
| `«s=N»` | Source citation reference (source record ID) |

### Cache Files

Cache files share a common header format: `size(4) + magic(4) + count(4)`, followed by format-specific data. They are regenerated by Reunion via File → Rebuild Cache Files, so they are redundant to `familyfile.familydata` but provide fast lookup indices.

#### `places.cache` (magic: `"ahcp"`)

Contains full-length place names. The `familydata` file truncates long place names, so this cache provides the complete versions.

```
Header:  size(4) + "ahcp"(4) + count(4) + extra(4) = 16 bytes
         count × uint32 offset table (starting at byte 16)

Each record (at offset):
         size(4) + id(4) + ref(8) + UTF-8 place name string
```

#### `placeUsage.cache` (magic: `"hcup"`)

Cross-references places to events:

```
Header:  size(4) + "hcup"(4) + count(4) + extra(4) = 16 bytes, then 4-byte sub-header

Each record:
         total_size(4) + n_entries(4) + place_id(4) + zero(4)
         + [ref_id(4) + type_code(4)] × n_entries
```

#### `fmnames.cache` (magic: `"2wps"`)

Given/first name index:

```
Header:  size(4) + "2wps"(4) + count(4) = 12 bytes
         count × uint32 offset table (starting at byte 12)

Each record:
         size(1) + meta(5) + phonetic(2) + name string
```

#### `surnames.cache` (magic: `"10ns"`)

Surname index stored as parenthesized entries like `(SURNAME, GIVEN)` separated by binary delimiters.

### What's Not Yet Understood

| Area | Status |
|------|--------|
| Full set of person field tags (e.g. flags, checkboxes) | Partially known |
| Date precision flag for "before" qualifier | Unknown (`precFlags` values for "after" and "about" are known) |
| Event sub-header bytes 0-3 | Purpose unknown (not date-related; does not change when date changes) |
| Media metadata field encoding | Unknown |
| Doc (`0x2108`) and Report (`0x210C`) record internals | Unknown |
| 8-byte `ref` field semantics in place records | Unknown |
| `.changes` files in member directories | Unknown |
| `associations.cache` full structure | Unknown |
| `globalRecords.cache`, `bookmarks.cache` detailed format | Placeholder only |
