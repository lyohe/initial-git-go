//go:build show_diff
// +build show_diff

package main

import (
	"fmt"
	"os"
	"os/exec"
)

const (
	mtimeChanged = 0x0001
	ctimeChanged = 0x0002
	ownerChanged = 0x0004
	modeChanged  = 0x0008
	inodeChanged = 0x0010
	dataChanged  = 0x0020
)

func match_stat(ce *cacheEntry, st statInfo) uint32 {
	var changed uint32
	if ce.mtime.sec != st.mtime.sec || ce.mtime.nsec != st.mtime.nsec {
		changed |= mtimeChanged
	}
	if ce.ctime.sec != st.ctime.sec || ce.ctime.nsec != st.ctime.nsec {
		changed |= ctimeChanged
	}
	if ce.stUID != st.uid || ce.stGID != st.gid {
		changed |= ownerChanged
	}
	if ce.stMode != st.mode {
		changed |= modeChanged
	}
	if ce.stDev != st.dev || ce.stIno != st.ino {
		changed |= inodeChanged
	}
	if ce.stSize != st.size {
		changed |= dataChanged
	}
	return changed
}

func show_differences(ce *cacheEntry, oldContents []byte) {
	cmd := exec.Command("diff", "-u", "-", ce.name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return
	}
	if err := cmd.Start(); err != nil {
		return
	}
	_, _ = stdin.Write(oldContents)
	_ = stdin.Close()
	_ = cmd.Wait()
}

func main() {
	entries, err := read_cache()
	if err != nil {
		fmt.Fprintln(os.Stderr, "read_cache")
		os.Exit(1)
	}
	for i := 0; i < entries; i++ {
		ce := activeCache[i]
		st, err := stat_file(ce.name)
		if err != nil {
			fmt.Printf("%s: %s\n", ce.name, err)
			continue
		}
		changed := match_stat(ce, st)
		if changed == 0 {
			fmt.Printf("%s: ok\n", ce.name)
			continue
		}
		fmt.Printf("%s:  %s\n", ce.name, sha1_to_hex(ce.sha1[:]))
		oldContents, _, err := read_sha1_file(ce.sha1[:])
		if err != nil {
			continue
		}
		show_differences(ce, oldContents)
	}
}
