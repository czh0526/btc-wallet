package keystore

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type entryHeader byte

const (
	addrCommentHeader entryHeader = 1 << iota
	txCommentHeader
	deletedHeader
	scriptHeader
	addrHeader entryHeader = 0
)

type varEntries struct {
	store   *Store
	entries []io.WriterTo
}

func (v *varEntries) WriteTo(w io.Writer) (n int64, err error) {
	ss := v.entries

	var written int64
	for _, s := range ss {
		var err error
		if written, err = s.WriteTo(w); err != nil {
			return n + written, err
		}
		n += written
	}

	return n, nil
}

func (v *varEntries) ReadFrom(r io.Reader) (n int64, err error) {
	var read int64

	v.entries = nil
	wts := v.entries

	for {
		var header entryHeader
		if read, err = binaryRead(r, binary.LittleEndian, &header); err != nil {
			if err == io.EOF {
				return n + read, nil
			}
			return n + read, err
		}
		n += read

		var wt io.WriterTo
		switch header {
		case addrHeader:
			var entry addrEntry
			entry.addr.store = v.store
			if read, err = entry.ReadFrom(r); err != nil {
				return n + read, err
			}
			n += read
			wt = &entry

		case scriptHeader:
			var entry scriptEntry
			entry.script.store = v.store
			if read, err = entry.ReadFrom(r); err != nil {
				return n + read, err
			}
			n += read
			wt = &entry

		default:
			return n, fmt.Errorf("unknown entry header: %d", uint8(header))
		}

		if wt != nil {
			wts = append(wts, wt)
			v.entries = wts
		}
	}
}

func binaryRead(r io.Reader, order binary.ByteOrder, data interface{}) (n int64, err error) {
	var read int
	buf := make([]byte, binary.Size(data))
	if read, err = io.ReadFull(r, buf); err != nil {
		return int64(read), err
	}
	return int64(read), binary.Read(bytes.NewBuffer(buf), order, data)
}

func binaryWrite(w io.Writer, order binary.ByteOrder, data interface{}) (n int64, err error) {
	buf := bytes.Buffer{}
	if err = binary.Write(&buf, order, data); err != nil {
		return 0, err
	}

	written, err := w.Write(buf.Bytes())
	return int64(written), err
}
