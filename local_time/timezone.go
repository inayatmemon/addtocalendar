package local_time

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Parse "zoneinfo" time zone file.
// This is a fairly standard file format used on OS X, Linux, BSD, Sun, and others.
// See tzfile(5), https://en.wikipedia.org/wiki/Zoneinfo,
// and ftp://munnari.oz.au/pub/oldtz/

import (
	"errors"
	"syscall"
)

// registerLoadFromEmbeddedTZData is called by the time/tzdata package,
// if it is imported.
func registerLoadFromEmbeddedTZData(f func(string) (string, error)) {
	loadFromEmbeddedTZData = f
}

// loadFromEmbeddedTZData is used to load a specific tzdata file
// from tzdata information embedded in the binary itself.
// This is set when the time/tzdata package is imported,
// via registerLoadFromEmbeddedTzdata.
var loadFromEmbeddedTZData func(zipname string) (string, error)

// maxFileSize is the max permitted size of files read by readFile.
// As reference, the zoneinfo.zip distributed by Go is ~350 KB,
// so 10MB is overkill.
const maxFileSize = 10 << 20

type fileSizeError string

func (f fileSizeError) Error() string {
	return "time: file " + string(f) + " is too large"
}

// Copies of io.Seek* constants to avoid importing "io":
const (
	seekStart   = 0
	seekCurrent = 1
	seekEnd     = 2
)

// Simple I/O interface to binary blob of data.
type dataIO struct {
	p     []byte
	error bool
}

func (d *dataIO) read(n int) []byte {
	if len(d.p) < n {
		d.p = nil
		d.error = true
		return nil
	}
	p := d.p[0:n]
	d.p = d.p[n:]
	return p
}

func (d *dataIO) big4() (n uint32, ok bool) {
	p := d.read(4)
	if len(p) < 4 {
		d.error = true
		return 0, false
	}
	return uint32(p[3]) | uint32(p[2])<<8 | uint32(p[1])<<16 | uint32(p[0])<<24, true
}

func (d *dataIO) big8() (n uint64, ok bool) {
	n1, ok1 := d.big4()
	n2, ok2 := d.big4()
	if !ok1 || !ok2 {
		d.error = true
		return 0, false
	}
	return (uint64(n1) << 32) | uint64(n2), true
}

func (d *dataIO) byte() (n byte, ok bool) {
	p := d.read(1)
	if len(p) < 1 {
		d.error = true
		return 0, false
	}
	return p[0], true
}

// read returns the read of the data in the buffer.
func (d *dataIO) rest() []byte {
	r := d.p
	d.p = nil
	return r
}

// Make a string by stopping at the first NUL
func byteString(p []byte) string {
	for i := 0; i < len(p); i++ {
		if p[i] == 0 {
			return string(p[0:i])
		}
	}
	return string(p)
}

var badData = errors.New("malformed time zone information")

// loadTzinfoFromDirOrZip returns the contents of the file with the given name
// in dir. dir can either be an uncompressed zip file, or a directory.
func loadTzinfoFromDirOrZip(dir, name string) ([]byte, error) {
	if len(dir) > 4 && dir[len(dir)-4:] == ".zip" {
		return LoadTzinfoFromZip(dir, name)
	}
	if dir != "" {
		name = dir + "/" + name
	}
	return readFile(name)
}

// There are 500+ zoneinfo files. Rather than distribute them all
// individually, we ship them in an uncompressed zip file.
// Used this way, the zip file format serves as a commonly readable
// container for the individual small files. We choose zip over tar
// because zip files have a contiguous table of contents, making
// individual file lookups faster, and because the per-file overhead
// in a zip file is considerably less than tar's 512 bytes.

// get4 returns the little-endian 32-bit value in b.
func get4(b []byte) int {
	if len(b) < 4 {
		return 0
	}
	return int(b[0]) | int(b[1])<<8 | int(b[2])<<16 | int(b[3])<<24
}

// get2 returns the little-endian 16-bit value in b.
func get2(b []byte) int {
	if len(b) < 2 {
		return 0
	}
	return int(b[0]) | int(b[1])<<8
}

func open(name string) (uintptr, error) {
	fd, err := syscall.Open(name, syscall.O_RDONLY, 0)
	if err != nil {
		return 0, err
	}
	return uintptr(fd), nil
}

func closefd(fd uintptr) {
	syscall.Close(syscall.Handle(fd))
}

func preadn(fd uintptr, buf []byte, off int) error {
	whence := seekStart
	if off < 0 {
		whence = seekEnd
	}
	if _, err := syscall.Seek(syscall.Handle(fd), int64(off), whence); err != nil {
		return err
	}
	for len(buf) > 0 {
		m, err := syscall.Read(syscall.Handle(fd), buf)
		if m <= 0 {
			if err == nil {
				return errors.New("short read")
			}
			return err
		}
		buf = buf[m:]
	}
	return nil
}

// loadTzinfoFromZip returns the contents of the file with the given name
// in the given uncompressed zip file.
func LoadTzinfoFromZip(zipfile, name string) ([]byte, error) {
	fd, err := open(zipfile)
	if err != nil {
		return nil, err
	}
	defer closefd(fd)

	const (
		zecheader = 0x06054b50
		zcheader  = 0x02014b50
		ztailsize = 22

		zheadersize = 30
		zheader     = 0x04034b50
	)

	buf := make([]byte, ztailsize)
	if err := preadn(fd, buf, -ztailsize); err != nil || get4(buf) != zecheader {
		return nil, errors.New("corrupt zip file " + zipfile)
	}
	n := get2(buf[10:])
	size := get4(buf[12:])
	off := get4(buf[16:])

	buf = make([]byte, size)
	if err := preadn(fd, buf, off); err != nil {
		return nil, errors.New("corrupt zip file " + zipfile)
	}

	for i := 0; i < n; i++ {
		// zip entry layout:
		//	0	magic[4]
		//	4	madevers[1]
		//	5	madeos[1]
		//	6	extvers[1]
		//	7	extos[1]
		//	8	flags[2]
		//	10	meth[2]
		//	12	modtime[2]
		//	14	moddate[2]
		//	16	crc[4]
		//	20	csize[4]
		//	24	uncsize[4]
		//	28	namelen[2]
		//	30	xlen[2]
		//	32	fclen[2]
		//	34	disknum[2]
		//	36	iattr[2]
		//	38	eattr[4]
		//	42	off[4]
		//	46	name[namelen]
		//	46+namelen+xlen+fclen - next header
		//
		if get4(buf) != zcheader {
			break
		}
		meth := get2(buf[10:])
		size := get4(buf[24:])
		namelen := get2(buf[28:])
		xlen := get2(buf[30:])
		fclen := get2(buf[32:])
		off := get4(buf[42:])
		zname := buf[46 : 46+namelen]
		buf = buf[46+namelen+xlen+fclen:]
		if string(zname) != name {
			continue
		}
		if meth != 0 {
			return nil, errors.New("unsupported compression for " + name + " in " + zipfile)
		}

		// zip per-file header layout:
		//	0	magic[4]
		//	4	extvers[1]
		//	5	extos[1]
		//	6	flags[2]
		//	8	meth[2]
		//	10	modtime[2]
		//	12	moddate[2]
		//	14	crc[4]
		//	18	csize[4]
		//	22	uncsize[4]
		//	26	namelen[2]
		//	28	xlen[2]
		//	30	name[namelen]
		//	30+namelen+xlen - file data
		//
		buf = make([]byte, zheadersize+namelen)
		if err := preadn(fd, buf, off); err != nil ||
			get4(buf) != zheader ||
			get2(buf[8:]) != meth ||
			get2(buf[26:]) != namelen ||
			string(buf[30:30+namelen]) != name {
			return nil, errors.New("corrupt zip file " + zipfile)
		}
		xlen = get2(buf[28:])

		buf = make([]byte, size)
		if err := preadn(fd, buf, off+30+namelen+xlen); err != nil {
			return nil, errors.New("corrupt zip file " + zipfile)
		}

		return buf, nil
	}

	return nil, syscall.ENOENT
}

// loadTzinfoFromTzdata returns the time zone information of the time zone
// with the given name, from a tzdata database file as they are typically
// found on android.
var loadTzinfoFromTzdata func(file, name string) ([]byte, error)

// loadTzinfo returns the time zone information of the time zone
// with the given name, from a given source. A source may be a
// timezone database directory, tzdata database file or an uncompressed
// zip file, containing the contents of such a directory.
func loadTzinfo(name string, source string) ([]byte, error) {
	if len(source) >= 6 && source[len(source)-6:] == "tzdata" {
		return loadTzinfoFromTzdata(source, name)
	}
	return loadTzinfoFromDirOrZip(source, name)
}

func read(fd uintptr, buf []byte) (int, error) {
	return syscall.Read(syscall.Handle(fd), buf)
}

// readFile reads and returns the content of the named file.
// It is a trivial implementation of os.ReadFile, reimplemented
// here to avoid depending on io/ioutil or os.
// It returns an error if name exceeds maxFileSize bytes.
func readFile(name string) ([]byte, error) {
	f, err := open(name)
	if err != nil {
		return nil, err
	}
	defer closefd(f)
	var (
		buf [4096]byte
		ret []byte
		n   int
	)
	for {
		n, err = read(f, buf[:])
		if n > 0 {
			ret = append(ret, buf[:n]...)
		}
		if n == 0 || err != nil {
			break
		}
		if len(ret) > maxFileSize {
			return nil, fileSizeError(name)
		}
	}
	return ret, err
}
