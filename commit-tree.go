//go:build commit_tree
// +build commit_tree

package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/user"
	"time"
)

const maxParent = 16

func remove_special(s string) string {
	buf := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\n', '<', '>':
			continue
		default:
			buf = append(buf, s[i])
		}
	}
	return string(buf)
}

func main() {
	if len(os.Args) < 2 {
		usage("commit-tree <sha1> [-p <sha1>]* < changelog")
	}
	treeSHA1, err := get_sha1_hex(os.Args[1])
	if err != nil {
		usage("commit-tree <sha1> [-p <sha1>]* < changelog")
	}
	parents := make([][20]byte, 0, maxParent)
	for i := 2; i < len(os.Args); i += 2 {
		if i+1 >= len(os.Args) || os.Args[i] != "-p" {
			usage("commit-tree <sha1> [-p <sha1>]* < changelog")
		}
		if len(parents) >= maxParent {
			usage("commit-tree <sha1> [-p <sha1>]* < changelog")
		}
		parent, err := get_sha1_hex(os.Args[i+1])
		if err != nil {
			usage("commit-tree <sha1> [-p <sha1>]* < changelog")
		}
		parents = append(parents, parent)
	}
	if len(parents) == 0 {
		fmt.Fprintf(os.Stderr, "Committing initial tree %s\n", os.Args[1])
	}

	u, err := user.Current()
	if err != nil {
		usage("You don't exist. Go away!")
	}
	realgecos := u.Name
	if realgecos == "" {
		realgecos = u.Username
	}
	hostname, _ := os.Hostname()
	realemail := fmt.Sprintf("%s@%s", u.Username, hostname)
	realdate := time.Now().Format(time.ANSIC)

	gecos := os.Getenv("COMMITTER_NAME")
	if gecos == "" {
		gecos = realgecos
	}
	email := os.Getenv("COMMITTER_EMAIL")
	if email == "" {
		email = realemail
	}
	date := os.Getenv("COMMITTER_DATE")
	if date == "" {
		date = realdate
	}

	gecos = remove_special(gecos)
	realgecos = remove_special(realgecos)
	email = remove_special(email)
	realemail = remove_special(realemail)
	date = remove_special(date)
	realdate = remove_special(realdate)

	var payload bytes.Buffer
	fmt.Fprintf(&payload, "tree %s\n", sha1_to_hex(treeSHA1[:]))
	for _, parent := range parents {
		fmt.Fprintf(&payload, "parent %s\n", sha1_to_hex(parent[:]))
	}
	fmt.Fprintf(&payload, "author %s <%s> %s\n", gecos, email, date)
	fmt.Fprintf(&payload, "committer %s <%s> %s\n\n", realgecos, realemail, realdate)

	comment, _ := io.ReadAll(os.Stdin)
	payload.Write(comment)

	header := fmt.Sprintf("commit %d", payload.Len())
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
