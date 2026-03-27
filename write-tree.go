//go:build write_tree
// +build write_tree

package main

import (
	"bytes"
	"fmt"
	"os"
)

func check_valid_sha1(sha1Bytes []byte) error {
	filename := sha1_file_name(sha1Bytes)
	if _, err := os.Stat(filename); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}

func main() {
	entries, err := read_cache()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if entries <= 0 {
		fmt.Fprintln(os.Stderr, "No file-cache to create a tree of")
		os.Exit(1)
	}

	var payload bytes.Buffer
	for _, ce := range activeCache {
		if err := check_valid_sha1(ce.sha1[:]); err != nil {
			os.Exit(1)
		}
		fmt.Fprintf(&payload, "%o %s", ce.stMode, ce.name)
		payload.WriteByte(0)
		payload.Write(ce.sha1[:])
	}

	header := fmt.Sprintf("tree %d", payload.Len())
	buf := make([]byte, 0, len(header)+1+payload.Len())
	buf = append(buf, header...)
	buf = append(buf, 0)
	buf = append(buf, payload.Bytes()...)

	sum, err := write_sha1_file(buf)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(sha1_to_hex(sum[:]))
}
