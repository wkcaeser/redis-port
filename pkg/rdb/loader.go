package rdb

import (
	"encoding/binary"
	"io"
	"strconv"

	"github.com/CodisLabs/codis/pkg/utils/log"
)

type Loader struct {
	r io.Reader

	rio redisRio

	header struct {
		version int64 // rdb version
	}
	cursor struct {
		db       uint64 // current database
		checksum uint64 // current checksum
		offset   int64  // current offset of the underlying reader
	}
	footer struct {
		checksum uint64 // expected checksum
	}
}

func NewLoader(r io.Reader) *Loader {
	if r == nil {
		log.Panicf("Create loader with nil reader.")
	}
	l := &Loader{r: r}
	l.rio.init()
	return l
}

func (l *Loader) onRead(b []byte) int {
	n, err := l.r.Read(b)
	if err != nil {
		log.PanicErrorf(err, "Read bytes failed.")
	}
	l.cursor.offset += int64(n)
	return n
}

func (l *Loader) onWrite(b []byte) int {
	log.Panicf("Doesn't support write operation.")
	return 0
}

func (l *Loader) onTell() int64 {
	return l.cursor.offset
}

func (l *Loader) onFlush() int {
	log.Panicf("Doesn't support flush operation.")
	return 0
}

func (l *Loader) onUpdateChecksum(checksum uint64) {
	l.cursor.checksum = checksum
}

func (l *Loader) Header() {
	header := make([]byte, 9)
	if err := l.rio.Read(header); err != nil {
		log.PanicErrorf(err, "Read RDB header failed.")
	}
	if format := string(header[:5]); format != "REDIS" {
		log.Panicf("Verify magic string, invalid format = '%s'.", format)
	}
	n, err := strconv.ParseInt(string(header[5:]), 10, 64)
	if err != nil {
		log.PanicErrorf(err, "Try to parse version = '%s'.", header[5:])
	}
	switch {
	case n < 1 || n > RDB_VERSION:
		log.Panicf("Can't handle RDB format version = %d.", n)
	default:
		l.header.version = n
	}
}

func (l *Loader) Footer() {
	var expected = l.cursor.checksum
	if l.header.version >= 5 {
		footer := make([]byte, 8)
		if err := l.rio.Read(footer); err != nil {
			log.PanicErrorf(err, "Read RDB footer failed.")
		}
		l.footer.checksum = binary.LittleEndian.Uint64(footer)
		switch {
		case l.footer.checksum == 0:
			log.Debugf("RDB file was saved with checksum disabled.")
		case l.footer.checksum != expected:
			log.Panicf("Wrong checksum, expected = %#16x, footer = %#16x.", expected, l.footer.checksum)
		}
	}
}

func (l *Loader) Next() interface{} {
	panic("TODO")
}
