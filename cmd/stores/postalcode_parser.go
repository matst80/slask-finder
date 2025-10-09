package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Stream parser for Swedish postal code CSV files with header:
//
// Postnummer,Ort,KnNamn,KnKod,LnNamn,Latitude,Longitude,Google-maps
//
// We only care about: Postnummer, Ort, Latitude, Longitude
//
// The Google-maps column (and others) are ignored. The parser works in a
// streaming fashion so large files can be processed without loading
// everything into memory.
//
// Typical usage:
//
//   f, _ := os.Open("postnummer.csv")
//   defer f.Close()
//   ctx := context.Background()
//   ch, errCh := PostalCodeLocationChannel(ctx, f)
//   for p := range ch {
//       // use p
//   }
//   if err := <-errCh; err != nil {
//       log.Fatalf("stream failed: %v", err)
//   }
//
// Or using the functional emitter form:
//
//   err := StreamPostalCodeLocations(ctx, f, func(p PostalCodeLocation) error {
//       // handle p
//       return nil
//   })
//
// The parser attempts to be resilient: lines with parsing issues are skipped.
// A final non-nil error indicates a fatal issue (e.g. bad header / read error).
//
// Postal codes are normalized by removing any internal spaces.
//
// NOTE: This file depends on types defined in postalcode.go and types.go
// (PostalCodeLocation and Location) within the same package.

var (
	// ErrMissingColumns is returned if required header columns are not found.
	ErrMissingColumns = errors.New("missing required postal code columns")
	// ErrEmptyInput indicates no data rows were found.
	ErrEmptyInput = errors.New("no postal code rows found")
)

// PostalCodeCSVConfig allows optional customization of the parser.
type PostalCodeCSVConfig struct {
	// Required names (case-insensitive, trimmed) - override if source differs.
	HeaderPostalCode string
	HeaderCity       string
	HeaderLatitude   string
	HeaderLongitude  string

	// If true, a duplicate postal code (same code + city) overwrites the earlier one.
	// If false, duplicates are all emitted in order of appearance.
	AllowOverwrite bool

	// CSV field delimiter (defaults to ','); set to ';' for Norwegian source.
	Delimiter rune
}

// SwedenPostalCodeCSVConfig returns the default (Swedish) header expectations.
func SwedenPostalCodeCSVConfig() PostalCodeCSVConfig {
	return PostalCodeCSVConfig{
		HeaderPostalCode: "postnummer",
		HeaderCity:       "ort",
		HeaderLatitude:   "latitude",
		HeaderLongitude:  "longitude",
		AllowOverwrite:   false,
		Delimiter:        ',',
	}
}

// NorwayPostalCodeCSVConfig returns configuration for Norwegian postal code CSV files.
// Expected header (semicolon separated):
// Postnummer;Poststed;FylkeKode;Fylke;KommuneKode;Kommune;PostnummerKategoriKode;PostnummerKategori;Latitude;Longitude
func NorwayPostalCodeCSVConfig() PostalCodeCSVConfig {
	return PostalCodeCSVConfig{
		HeaderPostalCode: "postnummer",
		HeaderCity:       "poststed",
		HeaderLatitude:   "latitude",
		HeaderLongitude:  "longitude",
		AllowOverwrite:   false,
		Delimiter:        ';',
	}
}

// StreamPostalCodeLocations streams postal code locations from r, invoking emit for each record.
// Stops early if ctx is canceled or if emit returns an error.
// Returns a fatal error (e.g. header invalid, IO error) or nil.
func StreamPostalCodeLocations(ctx context.Context, r io.Reader, emit func(PostalCodeLocation) error) error {
	return StreamPostalCodeLocationsWithConfig(ctx, r, emit, SwedenPostalCodeCSVConfig())
}

// StreamPostalCodeLocationsWithConfig same as StreamPostalCodeLocations but using a custom config.
func StreamPostalCodeLocationsWithConfig(ctx context.Context, r io.Reader, emit func(PostalCodeLocation) error, cfg PostalCodeCSVConfig) error {
	reader := csv.NewReader(NewNormalizedLineReader(r))
	if cfg.Delimiter != 0 {
		reader.Comma = cfg.Delimiter
	} else {
		reader.Comma = ','
	}
	// Keep lazy reading; we rely on reader.Read() to parse each record.
	reader.FieldsPerRecord = -1 // allow variable columns (we validate needed ones)
	reader.ReuseRecord = true

	// Read header
	header, err := reader.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return ErrEmptyInput
		}
		return fmt.Errorf("read header: %w", err)
	}

	colMap, err := mapPostalCodeHeader(header, cfg)
	if err != nil {
		return err
	}

	seen := map[string]struct{}{}
	rowCount := 0

	for {
		// Cancellation check
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		record, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("read row %d: %w", rowCount+2, err) // +2 accounts for header + 1-based
		}
		rowCount++

		if len(record) == 0 {
			continue
		}
		// Skip lines that are effectively blank
		isBlank := true
		for _, f := range record {
			if strings.TrimSpace(f) != "" {
				isBlank = false
				break
			}
		}
		if isBlank {
			continue
		}

		loc, ok := extractPostalCodeLocation(record, colMap)
		if !ok {
			// Skip malformed / invalid row silently.
			continue
		}

		key := loc.PostalCode + "::" + loc.City
		if _, exists := seen[key]; exists && !cfg.AllowOverwrite {
			// duplicate and we do not allow overwrites, just emit again
		}
		if cfg.AllowOverwrite {
			seen[key] = struct{}{}
		}

		if err := emit(loc); err != nil {
			return err
		}
	}

	return nil
}

// PostalCodeLocationChannel provides a channel-based wrapper around the functional streaming parser.
// The errors channel will receive a single value (nil or error) once the producer is done.
// The output channel is closed when streaming completes or an error/cancellation occurs.
func PostalCodeLocationChannel(ctx context.Context, r io.Reader) (<-chan PostalCodeLocation, <-chan error) {
	out := make(chan PostalCodeLocation)
	errCh := make(chan error, 1)

	go func() {
		defer close(out)
		err := StreamPostalCodeLocations(ctx, r, func(p PostalCodeLocation) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case out <- p:
				return nil
			}
		})
		errCh <- err
		close(errCh)
	}()

	return out, errCh
}

// headerColumns holds indices of required columns.
type headerColumns struct {
	PostalCode int
	City       int
	Latitude   int
	Longitude  int
}

// mapPostalCodeHeader finds required columns in the header slice.
func mapPostalCodeHeader(header []string, cfg PostalCodeCSVConfig) (headerColumns, error) {
	idx := func(target string) int {
		target = strings.ToLower(strings.TrimSpace(target))
		for i, h := range header {
			hNorm := strings.ToLower(strings.TrimSpace(stripBOM(h)))
			if hNorm == target {
				return i
			}
		}
		return -1
	}

	cols := headerColumns{
		PostalCode: idx(cfg.HeaderPostalCode),
		City:       idx(cfg.HeaderCity),
		Latitude:   idx(cfg.HeaderLatitude),
		Longitude:  idx(cfg.HeaderLongitude),
	}

	if cols.PostalCode < 0 || cols.City < 0 || cols.Latitude < 0 || cols.Longitude < 0 {
		return headerColumns{}, ErrMissingColumns
	}
	return cols, nil
}

// extractPostalCodeLocation builds a PostalCodeLocation from a CSV record using mapped indices.
// Returns (loc, true) on success; (_, false) on parse failure.
func extractPostalCodeLocation(record []string, cols headerColumns) (PostalCodeLocation, bool) {
	get := func(i int) string {
		if i >= 0 && i < len(record) {
			return strings.TrimSpace(record[i])
		}
		return ""
	}

	rawCode := normalizePostalCode(get(cols.PostalCode))
	if rawCode == "" {
		return PostalCodeLocation{}, false
	}

	city := strings.TrimSpace(get(cols.City))
	if city == "" {
		return PostalCodeLocation{}, false
	}

	lat, err := parseFloat(get(cols.Latitude), -90, 90)
	if err != nil {
		return PostalCodeLocation{}, false
	}
	lng, err := parseFloat(get(cols.Longitude), -180, 180)
	if err != nil {
		return PostalCodeLocation{}, false
	}

	return PostalCodeLocation{
		PostalCode: rawCode,
		City:       city,
		Location: Location{
			Latitude:  lat,
			Longitude: lng,
		},
	}, true
}

// normalizePostalCode removes whitespace inside the code.
func normalizePostalCode(code string) string {
	code = strings.TrimSpace(code)
	code = strings.ReplaceAll(code, " ", "")
	return code
}

// parseFloat parses a float64 and ensures it is within [min, max].
func parseFloat(s string, min, max float64) (float64, error) {
	if s == "" {
		return 0, errors.New("empty")
	}
	v, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", "."), 64)
	if err != nil {
		return 0, err
	}
	if v < min || v > max {
		return 0, fmt.Errorf("out of range: %v", v)
	}
	return v, nil
}

// NewNormalizedLineReader returns an io.Reader that strips a UTF-8 BOM
// if present in the very first line and leaves everything else unchanged.
// This helps when CSV files include a BOM in the first header cell.
func NewNormalizedLineReader(r io.Reader) io.Reader {
	br := bufio.NewReader(r)
	// Peek first 3 bytes to detect BOM
	b, err := br.Peek(3)
	if err == nil && len(b) == 3 && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF {
		// Discard BOM
		_, _ = br.Discard(3)
	}
	return br
}

// stripBOM removes a leading UTF-8 BOM from a string (header cell safety).
func stripBOM(s string) string {
	if len(s) >= 3 && s[0] == 0xEF && s[1] == 0xBB && s[2] == 0xBF {
		return s[3:]
	}
	return s
}
