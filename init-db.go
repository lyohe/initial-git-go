//go:build init_db
// +build init_db

package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	if err := os.Mkdir(".dircache", 0o700); err != nil {
		fmt.Fprintf(os.Stderr, "unable to create .dircache: %v\n", err)
		os.Exit(1)
	}

	sha1Dir := os.Getenv(dbEnvironment)
	if sha1Dir != "" {
		fmt.Fprintf(os.Stderr, "DB_ENVIRONMENT set to bad directory %s: ", sha1Dir)
	}

	sha1Dir = defaultDBEnvironment
	fmt.Fprintln(os.Stderr, "defaulting to private storage area")
	if err := os.Mkdir(sha1Dir, 0o700); err != nil && !os.IsExist(err) {
		fmt.Fprintln(os.Stderr, sha1Dir, err)
		os.Exit(1)
	}
	for i := 0; i < 256; i++ {
		path := filepath.Join(sha1Dir, fmt.Sprintf("%02x", i))
		if err := os.Mkdir(path, 0o700); err != nil && !os.IsExist(err) {
			fmt.Fprintln(os.Stderr, path, err)
			os.Exit(1)
		}
	}
}
