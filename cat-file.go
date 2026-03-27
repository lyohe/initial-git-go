//go:build cat_file
// +build cat_file

package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		usage("cat-file: cat-file <sha1>")
	}
	sha1Bytes, err := get_sha1_hex(os.Args[1])
	if err != nil {
		usage("cat-file: cat-file <sha1>")
	}
	buf, typ, err := read_sha1_file(sha1Bytes[:])
	if err != nil {
		os.Exit(1)
	}
	fd, err := os.CreateTemp(".", "temp_git_file_")
	if err != nil {
		usage("unable to create tempfile")
	}
	defer fd.Close()
	if _, err := fd.Write(buf); err != nil {
		typ = "bad"
	}
	fmt.Printf("%s: %s\n", fd.Name(), typ)
}
