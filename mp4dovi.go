package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
)

type FourCC [4]byte
type BoxType FourCC

var (
	MoovBoxType = BoxType{'m', 'o', 'o', 'v'}
	TrakBoxType = BoxType{'t', 'r', 'a', 'k'}
	MdiaBoxType = BoxType{'m', 'd', 'i', 'a'}
	MinfBoxType = BoxType{'m', 'i', 'n', 'f'}
	StblBoxType = BoxType{'s', 't', 'b', 'l'}
	StsdBoxType = BoxType{'s', 't', 's', 'd'}
)

type Header struct {
	Size uint32
	Type BoxType

	// Present only if Size == 1
	ExtendedSize uint64
}

var codecFrom string
var codecTo string
var verbose bool

func getBoxSize(header *Header) uint64 {
	if header.Size == 1 {
		return header.ExtendedSize
	}
	return uint64(header.Size)
}

func getHeaderSize(header *Header) uint64 {
	if header.Size == 1 {
		return 16
	}
	return 8
}

func getHeaderTypeOffset(header *Header) int64 {
	if header.Size == 1 {
		return -12
	}
	return -4
}

func readBoxHeader(r io.ReadSeeker) (*Header, error) {
	var header Header
	var err error
	err = binary.Read(r, binary.BigEndian, &header.Size)
	if err != nil {
		return nil, err
	}
	err = binary.Read(r, binary.BigEndian, &header.Type)
	if err != nil {
		return nil, err
	}

	if header.Size == 1 {
		err = binary.Read(r, binary.BigEndian, &header.ExtendedSize)
		if err != nil {
			return nil, err
		}
	}

	return &header, nil
}

func findHeader(r io.ReadSeeker, boxType BoxType, limit int64) (header *Header, err error) {
	var h *Header
	for offset := int64(0); limit < 0 || offset < limit; offset += int64(getBoxSize(h)) {
		if h, err = readBoxHeader(r); err != nil {
			return nil, fmt.Errorf(`[findHeader] failed reading box header: %w`, err)
		}

		if verbose {
			fmt.Printf("[findHeader] inspecting %s at %d(%#x)\n", string(h.Type[:]), offset, offset)
		}

		if h.Type == boxType {
			if verbose {
				fmt.Printf("[findHeader] found %s at %d(%#x)\n", string(h.Type[:]), offset, offset)
			}
			return h, nil
		}
		if _, err = r.Seek(int64(getBoxSize(h)-getHeaderSize(h)), io.SeekCurrent); err != nil {
			return nil, fmt.Errorf(`[findHeader] failed seeking after box "%s": %s`, h.Type, err)
		}
	}
	return nil, fmt.Errorf(`[findHeader] cannot find box "%s"`, boxType)
}

func forEachBox(r io.ReadSeeker, limit int64, fn func(header *Header) error) (err error) {
	var h *Header
	var start int64
	if start, err = r.Seek(0, io.SeekCurrent); err != nil {
		return fmt.Errorf(`[forEachBox] failed to get current offset with seek: %w`, err)
	}
	for offset := start; limit < 0 || offset < start+limit; offset += int64(getBoxSize(h)) {
		if _, err = r.Seek(offset, io.SeekStart); err != nil {
			return fmt.Errorf(`[forEachBox] failed to seek to offset: %w`, err)
		}

		if h, err = readBoxHeader(r); err != nil {
			return fmt.Errorf(`[forEachBox] failed reading box header: %w`, err)
		}

		if verbose {
			fmt.Printf("[forEachBox] inspecting %s at %d(%#x)\n", string(h.Type[:]), offset, offset)
		}

		if err = fn(h); err != nil {
			return fmt.Errorf(`[forEachBox] callback failed: %w`, err)
		}
	}
	return
}

func sampleEntryHandler(rw *os.File) func(*Header) error {
	return func(h *Header) (err error) {
		if string(h.Type[:]) == codecFrom {
			if _, err = rw.Seek(getHeaderTypeOffset(h), io.SeekCurrent); err != nil {
				return fmt.Errorf(`[sampleEntryHandler] failed to seek back: %w`, err)
			}
			if err = binary.Write(rw, binary.BigEndian, []byte(codecTo)); err != nil {
				return fmt.Errorf(`[sampleEntryHandler] failed to write box header type "%s": %w`, codecTo, err)
			}
			fmt.Printf("Changed codec from %v to %v\n", codecFrom, codecTo)
		}
		return
	}
}

func trakHandler(rw *os.File) func(*Header) error {
	return func(trak *Header) (err error) {
		var h *Header

		if trak.Type != TrakBoxType {
			return
		}

		if h, err = findHeader(rw, MdiaBoxType, int64(getBoxSize(trak)-getHeaderSize(trak))); err != nil {
			return fmt.Errorf(`[trakHandler] failed finding box "%s": %w`, MdiaBoxType, err)
		}

		if h, err = findHeader(rw, MinfBoxType, int64(getBoxSize(h)-getHeaderSize(h))); err != nil {
			return fmt.Errorf(`[trakHandler] failed finding box "%s": %w`, MinfBoxType, err)
		}

		if h, err = findHeader(rw, StblBoxType, int64(getBoxSize(h)-getHeaderSize(h))); err != nil {
			return fmt.Errorf(`[trakHandler] failed finding box "%s": %w`, StblBoxType, err)
		}

		if h, err = findHeader(rw, StsdBoxType, int64(getBoxSize(h)-getHeaderSize(h))); err != nil {
			return fmt.Errorf(`[trakHandler] failed finding box "%s": %w`, StsdBoxType, err)
		}

		// skip Version(1 byte) + Flags(3 bytes) + Number of entries(4 bytes) in stsd
		if _, err = rw.Seek(8, io.SeekCurrent); err != nil {
			return fmt.Errorf(`[trakHandler] failed to seek: %w`, err)
		}

		if err = forEachBox(rw, int64(getBoxSize(h)-getHeaderSize(h)), sampleEntryHandler(rw)); err != nil {
			return fmt.Errorf(`[trakHandler] failed processing sample entry list: %w`, err)
		}

		return
	}
}

func processFile(mp4file string) (err error) {
	var (
		rw *os.File
		h  *Header
	)

	if rw, err = os.OpenFile(mp4file, os.O_RDWR, 0); err != nil {
		return fmt.Errorf(`[processFile] cannot open file "%s": %w`, mp4file, err)
	}
	defer func(rw *os.File) {
		filename := rw.Name()
		err := rw.Close()
		if err != nil {
			_ = fmt.Errorf("[processFile] cannot close file %v", filename)
		}
	}(rw)

	fmt.Printf("Processing %s ...\n", mp4file)

	if _, err = rw.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf(`[processFile] failed to seek: %w`, err)
	}

	if h, err = findHeader(rw, MoovBoxType, -1); err != nil {
		return fmt.Errorf(`[processFile] failed finding box "%s": %w`, MoovBoxType, err)
	}

	if err = forEachBox(rw, int64(getBoxSize(h)-getHeaderSize(h)), trakHandler(rw)); err != nil {
		return fmt.Errorf(`[processFile] failed processing moov children: %w`, err)
	}
	return
}

func run(mp4files []string) (err error) {
	for _, mp4file := range mp4files {
		if err = processFile(mp4file); err != nil {
			return fmt.Errorf(`[run] failed processing file %s: %w`, mp4file, err)
		}
	}
	return
}

func help() {
	fmt.Printf("usage: mp4dovi [options] files...\n")
	flag.PrintDefaults()
}

func main() {
	flag.StringVar(&codecFrom, "from", "dvhe", "video codec to convert from")
	flag.StringVar(&codecTo, "to", "dvh1", "video codec to convert to")
	flag.BoolVar(&verbose, "verbose", false, "enable verbose output")
	flag.Parse()

	files := flag.Args()
	if len(files) < 1 {
		help()
		os.Exit(1)
	}

	if err := run(files); err != nil {
		log.Fatal(err)
	}
}
