//go:build update_cache
// +build update_cache

package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

func cache_name_compare(name1 string, name2 string) int {
	b1 := []byte(name1)
	b2 := []byte(name2)
	min := len(b1)
	if len(b2) < min {
		min = len(b2)
	}
	cmp := bytes.Compare(b1[:min], b2[:min])
	if cmp != 0 {
		return cmp
	}
	if len(b1) < len(b2) {
		return -1
	}
	if len(b1) > len(b2) {
		return 1
	}
	return 0
}

func cache_name_pos(name string) int {
	first := 0
	last := len(activeCache)
	for last > first {
		next := (last + first) >> 1
		ce := activeCache[next]
		cmp := cache_name_compare(name, ce.name)
		if cmp == 0 {
			return -next - 1
		}
		if cmp < 0 {
			last = next
			continue
		}
		first = next + 1
	}
	return first
}

func remove_file_from_cache(path string) {
	pos := cache_name_pos(path)
	if pos < 0 {
		idx := -pos - 1
		copy(activeCache[idx:], activeCache[idx+1:])
		activeCache = activeCache[:len(activeCache)-1]
	}
}

func add_cache_entry(ce *cacheEntry) {
	pos := cache_name_pos(ce.name)
	if pos < 0 {
		activeCache[-pos-1] = ce
		return
	}
	activeCache = append(activeCache, nil)
	copy(activeCache[pos+1:], activeCache[pos:])
	activeCache[pos] = ce
}

func add_file_to_cache(path string) error {
	fd, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			remove_file_from_cache(path)
			return nil
		}
		return err
	}
	defer fd.Close()

	st, err := stat_file(path)
	if err != nil {
		return err
	}
	content, err := io.ReadAll(fd)
	if err != nil {
		return err
	}

	ce := &cacheEntry{
		ctime: cacheTime{
			sec:  st.ctime.sec,
			nsec: st.ctime.nsec,
		},
		mtime: cacheTime{
			sec:  st.mtime.sec,
			nsec: st.mtime.nsec,
		},
		stDev:   st.dev,
		stIno:   st.ino,
		stMode:  st.mode,
		stUID:   st.uid,
		stGID:   st.gid,
		stSize:  st.size,
		namelen: uint16(len(path)),
		name:    path,
	}

	header := fmt.Sprintf("blob %d", len(content))
	buf := make([]byte, 0, len(header)+1+len(content))
	buf = append(buf, header...)
	buf = append(buf, 0)
	buf = append(buf, content...)
	sum, err := write_sha1_file(buf)
	if err != nil {
		return err
	}
	ce.sha1 = sum
	add_cache_entry(ce)
	return nil
}

func write_cache(fd *os.File, cache []*cacheEntry) error {
	header := make([]byte, 32)
	binary.LittleEndian.PutUint32(header[0:], cacheSignature)
	binary.LittleEndian.PutUint32(header[4:], cacheVersion)
	binary.LittleEndian.PutUint32(header[8:], uint32(len(cache)))

	hasher := sha1.New()
	hasher.Write(header[:12])
	entries := make([][]byte, len(cache))
	for i, ce := range cache {
		enc := encode_cache_entry(ce)
		entries[i] = enc
		hasher.Write(enc)
	}
	sum := hasher.Sum(nil)
	copy(header[12:], sum)

	if _, err := fd.Write(header); err != nil {
		return err
	}
	for _, entry := range entries {
		if _, err := fd.Write(entry); err != nil {
			return err
		}
	}
	return nil
}

func verify_path(path string) bool {
	if path == "" || path[len(path)-1] == '/' {
		return false
	}
	for i := 0; i < len(path); i++ {
		if i == 0 || path[i-1] == '/' {
			if path[i] == '/' || path[i] == '.' {
				return false
			}
		}
	}
	return true
}

func main() {
	_, err := read_cache()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cache corrupted")
		return
	}

	lock, err := os.OpenFile(".dircache/index.lock", os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to create new cachefile")
		return
	}
	defer lock.Close()

	for _, path := range os.Args[1:] {
		if !verify_path(path) {
			fmt.Fprintf(os.Stderr, "Ignoring path %s\n", path)
			continue
		}
		if err := add_file_to_cache(path); err != nil {
			fmt.Fprintf(os.Stderr, "Unable to add %s to database\n", path)
			goto out
		}
	}
	if err := write_cache(lock, activeCache); err == nil {
		lock.Close()
		if err := os.Rename(".dircache/index.lock", ".dircache/index"); err == nil {
			return
		}
	}
out:
	_ = os.Remove(".dircache/index.lock")
}
