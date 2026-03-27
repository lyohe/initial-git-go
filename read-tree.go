//go:build read_tree
// +build read_tree

package main

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
)

func unpack(sha1Bytes []byte) error {
	buffer, typ, err := read_sha1_file(sha1Bytes)
	if err != nil {
		return err
	}
	if typ != "tree" {
		return fmt.Errorf("expected a 'tree' node")
	}
	offset := 0
	for offset < len(buffer) {
		space := bytes.IndexByte(buffer[offset:], ' ')
		if space < 0 {
			return fmt.Errorf("corrupt 'tree' file")
		}
		modeStr := string(buffer[offset : offset+space])
		mode, err := strconv.ParseUint(modeStr, 8, 32)
		if err != nil {
			return fmt.Errorf("corrupt 'tree' file")
		}
		offset += space + 1
		nul := bytes.IndexByte(buffer[offset:], 0)
		if nul < 0 {
			return fmt.Errorf("corrupt 'tree' file")
		}
		path := string(buffer[offset : offset+nul])
		offset += nul + 1
		if offset+20 > len(buffer) {
			return fmt.Errorf("corrupt 'tree' file")
		}
		fmt.Printf("%o %s (%s)\n", mode, path, sha1_to_hex(buffer[offset:offset+20]))
		offset += 20
	}
	return nil
}

func main() {
	if len(os.Args) != 2 {
		usage("read-tree <key>")
	}
	sha1Bytes, err := get_sha1_hex(os.Args[1])
	if err != nil {
		usage("read-tree <key>")
	}
	sha1FileDirectory = os.Getenv(dbEnvironment)
	if sha1FileDirectory == "" {
		sha1FileDirectory = defaultDBEnvironment
	}
	if err := unpack(sha1Bytes[:]); err != nil {
		usage("unpack failed")
	}
}
