package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
)

func usage(err string) {
	fmt.Fprintf(os.Stderr, "read-tree: %s\n", err)
	os.Exit(1)
}

func hexval(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	default:
		return -1
	}
}

func get_sha1_hex(hexStr string) ([20]byte, error) {
	var sha1Bytes [20]byte
	if len(hexStr) != 40 {
		return sha1Bytes, fmt.Errorf("bad sha1 length")
	}
	for i := 0; i < 20; i++ {
		hi := hexval(hexStr[i*2])
		lo := hexval(hexStr[i*2+1])
		if hi < 0 || lo < 0 {
			return sha1Bytes, fmt.Errorf("bad sha1 hex")
		}
		sha1Bytes[i] = byte((hi << 4) | lo)
	}
	return sha1Bytes, nil
}

func sha1_to_hex(sha1Bytes []byte) string {
	const hex = "0123456789abcdef"
	if len(sha1Bytes) < 20 {
		return ""
	}
	out := make([]byte, 40)
	for i := 0; i < 20; i++ {
		val := sha1Bytes[i]
		out[i*2] = hex[val>>4]
		out[i*2+1] = hex[val&0x0f]
	}
	return string(out)
}

func sha1_file_name(sha1Bytes []byte) string {
	dir := sha1FileDirectory
	if dir == "" {
		dir = os.Getenv(dbEnvironment)
		if dir == "" {
			dir = defaultDBEnvironment
		}
	}
	hex := sha1_to_hex(sha1Bytes)
	if len(hex) < 2 {
		return ""
	}
	return filepath.Join(dir, hex[:2], hex[2:])
}

func read_sha1_file(sha1Bytes []byte) ([]byte, string, error) {
	path := sha1_file_name(sha1Bytes)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	zr, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, "", err
	}
	defer zr.Close()
	inflated, err := io.ReadAll(zr)
	if err != nil {
		return nil, "", err
	}
	nul := bytes.IndexByte(inflated, 0)
	if nul < 0 {
		return nil, "", fmt.Errorf("corrupt object header")
	}
	header := string(inflated[:nul])
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 {
		return nil, "", fmt.Errorf("corrupt object header")
	}
	size, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return nil, "", fmt.Errorf("corrupt object size")
	}
	content := inflated[nul+1:]
	if uint64(len(content)) < size {
		return nil, "", fmt.Errorf("object truncated")
	}
	return content[:size], parts[0], nil
}

func write_sha1_buffer(sha1Bytes []byte, compressed []byte) error {
	path := sha1_file_name(sha1Bytes)
	fd, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil
		}
		return err
	}
	defer fd.Close()
	_, err = fd.Write(compressed)
	return err
}

func write_sha1_file(buf []byte) ([20]byte, error) {
	var out [20]byte
	var compressed bytes.Buffer
	zw, err := zlib.NewWriterLevel(&compressed, zlib.BestCompression)
	if err != nil {
		return out, err
	}
	if _, err := zw.Write(buf); err != nil {
		zw.Close()
		return out, err
	}
	if err := zw.Close(); err != nil {
		return out, err
	}
	sum := sha1.Sum(compressed.Bytes())
	copy(out[:], sum[:])
	if err := write_sha1_buffer(out[:], compressed.Bytes()); err != nil {
		return out, err
	}
	return out, nil
}

func verify_hdr(data []byte, hdr cacheHeader) error {
	if hdr.signature != cacheSignature {
		return fmt.Errorf("bad signature")
	}
	if hdr.version != cacheVersion {
		return fmt.Errorf("bad version")
	}
	hash := sha1.New()
	hash.Write(data[:12])
	hash.Write(data[32:])
	sum := hash.Sum(nil)
	if !bytes.Equal(sum, hdr.sha1[:]) {
		return fmt.Errorf("bad header sha1")
	}
	return nil
}

func read_cache() (int, error) {
	if activeCache != nil {
		return 0, fmt.Errorf("more than one cachefile")
	}
	sha1FileDirectory = os.Getenv(dbEnvironment)
	if sha1FileDirectory == "" {
		sha1FileDirectory = defaultDBEnvironment
	}
	if err := ensure_dir_access(sha1FileDirectory); err != nil {
		return 0, err
	}
	data, err := os.ReadFile(".dircache/index")
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	if len(data) < 32 {
		return 0, fmt.Errorf("bad header")
	}
	hdr := parse_cache_header(data[:32])
	if err := verify_hdr(data, hdr); err != nil {
		return 0, err
	}
	entries := int(hdr.entries)
	activeCache = make([]*cacheEntry, 0, entries)
	offset := 32
	for i := 0; i < entries; i++ {
		ce, size, err := parse_cache_entry(data[offset:])
		if err != nil {
			activeCache = nil
			return 0, err
		}
		activeCache = append(activeCache, ce)
		offset += size
	}
	return len(activeCache), nil
}

func parse_cache_header(data []byte) cacheHeader {
	var hdr cacheHeader
	hdr.signature = binary.LittleEndian.Uint32(data[0:4])
	hdr.version = binary.LittleEndian.Uint32(data[4:8])
	hdr.entries = binary.LittleEndian.Uint32(data[8:12])
	copy(hdr.sha1[:], data[12:32])
	return hdr
}

func parse_cache_entry(data []byte) (*cacheEntry, int, error) {
	if len(data) < 62 {
		return nil, 0, fmt.Errorf("bad cache entry")
	}
	nameLen := int(binary.LittleEndian.Uint16(data[60:62]))
	entrySize := cacheEntrySize(nameLen)
	if len(data) < entrySize {
		return nil, 0, fmt.Errorf("bad cache entry")
	}
	if 62+nameLen > len(data) {
		return nil, 0, fmt.Errorf("bad cache entry")
	}
	name := string(data[62 : 62+nameLen])
	ce := &cacheEntry{
		ctime: cacheTime{
			sec:  binary.LittleEndian.Uint32(data[0:4]),
			nsec: binary.LittleEndian.Uint32(data[4:8]),
		},
		mtime: cacheTime{
			sec:  binary.LittleEndian.Uint32(data[8:12]),
			nsec: binary.LittleEndian.Uint32(data[12:16]),
		},
		stDev:   binary.LittleEndian.Uint32(data[16:20]),
		stIno:   binary.LittleEndian.Uint32(data[20:24]),
		stMode:  binary.LittleEndian.Uint32(data[24:28]),
		stUID:   binary.LittleEndian.Uint32(data[28:32]),
		stGID:   binary.LittleEndian.Uint32(data[32:36]),
		stSize:  binary.LittleEndian.Uint32(data[36:40]),
		namelen: uint16(nameLen),
		name:    name,
	}
	copy(ce.sha1[:], data[40:60])
	return ce, entrySize, nil
}

func encode_cache_entry(ce *cacheEntry) []byte {
	nameLen := int(ce.namelen)
	entrySize := cacheEntrySize(nameLen)
	buf := make([]byte, entrySize)
	binary.LittleEndian.PutUint32(buf[0:], ce.ctime.sec)
	binary.LittleEndian.PutUint32(buf[4:], ce.ctime.nsec)
	binary.LittleEndian.PutUint32(buf[8:], ce.mtime.sec)
	binary.LittleEndian.PutUint32(buf[12:], ce.mtime.nsec)
	binary.LittleEndian.PutUint32(buf[16:], ce.stDev)
	binary.LittleEndian.PutUint32(buf[20:], ce.stIno)
	binary.LittleEndian.PutUint32(buf[24:], ce.stMode)
	binary.LittleEndian.PutUint32(buf[28:], ce.stUID)
	binary.LittleEndian.PutUint32(buf[32:], ce.stGID)
	binary.LittleEndian.PutUint32(buf[36:], ce.stSize)
	copy(buf[40:], ce.sha1[:])
	binary.LittleEndian.PutUint16(buf[60:], ce.namelen)
	copy(buf[62:], []byte(ce.name))
	return buf
}

type statInfo struct {
	ctime cacheTime
	mtime cacheTime
	dev   uint32
	ino   uint32
	mode  uint32
	uid   uint32
	gid   uint32
	size  uint32
}

func stat_file(path string) (statInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return statInfo{}, err
	}
	mod := info.ModTime()
	st := statInfo{
		ctime: cacheTime{sec: uint32(mod.Unix()), nsec: uint32(mod.Nanosecond())},
		mtime: cacheTime{sec: uint32(mod.Unix()), nsec: uint32(mod.Nanosecond())},
		size:  uint32(info.Size()),
	}
	sys, ok := info.Sys().(*syscall.Stat_t)
	if !ok || sys == nil {
		return st, nil
	}
	st.dev = uint32(sys.Dev)
	st.ino = uint32(sys.Ino)
	st.mode = uint32(sys.Mode)
	st.uid = uint32(sys.Uid)
	st.gid = uint32(sys.Gid)
	st.size = uint32(sys.Size)

	ctimeSec, ctimeNsec := timespec_field(sys, []string{"Ctim", "Ctimespec"})
	mtimeSec, mtimeNsec := timespec_field(sys, []string{"Mtim", "Mtimespec"})
	if ctimeSec != 0 || ctimeNsec != 0 {
		st.ctime.sec = ctimeSec
		st.ctime.nsec = ctimeNsec
	}
	if mtimeSec != 0 || mtimeNsec != 0 {
		st.mtime.sec = mtimeSec
		st.mtime.nsec = mtimeNsec
	}
	return st, nil
}

func timespec_field(sys *syscall.Stat_t, names []string) (uint32, uint32) {
	v := reflect.ValueOf(sys).Elem()
	for _, name := range names {
		field := v.FieldByName(name)
		if !field.IsValid() {
			continue
		}
		sec := field.FieldByName("Sec")
		nsec := field.FieldByName("Nsec")
		if !sec.IsValid() || !nsec.IsValid() {
			continue
		}
		return uint32(sec.Int()), uint32(nsec.Int())
	}
	return 0, 0
}

func ensure_dir_access(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("no access to SHA1 file directory")
	}
	return nil
}
