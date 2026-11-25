package runs

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Limits for files read out of GitHub's zips. The zips themselves are
// limited by the GitHub client (spec §9).
const (
	// maxEntry caps an uncompressed file, so a small zip cannot expand
	// without bound.
	maxEntry = 64 << 20
	// maxRunLog caps the stored run log. A longer log keeps its end, where
	// failures are, behind a note.
	maxRunLog = 10 << 20
)

// errTooLarge is a file in a zip larger than maxEntry.
var errTooLarge = fmt.Errorf("larger than %d MiB", maxEntry>>20)

// firstXML returns the first *.xml file of a zip in archive order (legacy:
// fetchWorkflowResultsLog), or nil if there is none. A file larger than
// maxEntry is errTooLarge.
func firstXML(data []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("results artifact: %w", err)
	}
	for _, f := range zr.File {
		if f.FileInfo().IsDir() || !strings.HasSuffix(strings.ToLower(f.Name), ".xml") {
			continue
		}
		b, cut, err := readEntry(f)
		switch {
		case err != nil:
			return nil, fmt.Errorf("results artifact %s: %w", f.Name, err)
		case cut:
			return nil, fmt.Errorf("results artifact %s: %w", f.Name, errTooLarge)
		}
		return b, nil
	}
	return nil, nil
}

// largestFile returns the largest file of the run log zip as text: the log
// that combines all jobs (legacy: fetchWorkflowRunLog). An empty zip gives
// "". A log longer than maxRunLog keeps its end, where failures are, behind
// a note.
func largestFile(data []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("run log: %w", err)
	}
	var largest *zip.File
	for _, f := range zr.File {
		if !f.FileInfo().IsDir() && (largest == nil || f.UncompressedSize64 > largest.UncompressedSize64) {
			largest = f
		}
	}
	if largest == nil {
		return "", nil
	}
	b, cut, err := readEntry(largest)
	if err != nil {
		return "", fmt.Errorf("run log %s: %w", largest.Name, err)
	}
	if cut || len(b) > maxRunLog {
		b = append([]byte("[TestFlow: this log is too long; only its last "+strconv.Itoa(maxRunLog>>20)+" MiB are kept]\n"),
			b[max(len(b)-maxRunLog, 0):]...)
	}
	return text(b), nil
}

// readEntry reads a zip file, at most maxEntry bytes of it; cut reports
// that there was more.
func readEntry(f *zip.File) (b []byte, cut bool, err error) {
	rc, err := f.Open()
	if err != nil {
		return nil, false, err
	}
	defer rc.Close()
	b, err = io.ReadAll(io.LimitReader(rc, maxEntry+1))
	if err != nil {
		return nil, false, err
	}
	if len(b) > maxEntry {
		return b[:maxEntry], true, nil
	}
	return b, false, nil
}

// text makes b safe for a Postgres text column: invalid UTF-8 becomes
// U+FFFD and NUL bytes, which text cannot hold, are dropped.
func text(b []byte) string {
	return strings.ReplaceAll(strings.ToValidUTF8(string(b), "�"), "\x00", "")
}
