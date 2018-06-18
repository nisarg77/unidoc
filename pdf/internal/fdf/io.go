/*
 * This file is subject to the terms and conditions defined in
 * file 'LICENSE.md', which is part of this source code package.
 */

package fdf

import (
	"bufio"
	"errors"
	"os"

	"github.com/unidoc/unidoc/common"
	"io"
)

// readAtLeast reads at least n bytes into slice p.
// Returns the number of bytes read (should always be == n), and an error on failure.
func (parser *FdfParser) readAtLeast(p []byte, n int) (int, error) {
	remaining := n
	start := 0
	numRounds := 0
	for remaining > 0 {
		nRead, err := parser.reader.Read(p[start:])
		if err != nil {
			common.Log.Debug("ERROR Failed reading (%d;%d) %s", nRead, numRounds, err.Error())
			return start, errors.New("Failed reading")
		}
		numRounds++
		start += nRead
		remaining -= nRead
	}
	return start, nil
}

// getFileOffset returns the current file offset, accounting for buffered position.
func (parser *FdfParser) getFileOffset() int64 {
	offset, _ := parser.rs.Seek(0, os.SEEK_CUR)
	offset -= int64(parser.reader.Buffered())
	return offset
}

// setFileOffset seeks the file to an offset position.
func (parser *FdfParser) setFileOffset(offset int64) {
	parser.rs.Seek(offset, io.SeekStart)
	parser.reader = bufio.NewReader(parser.rs)
}