package main

const (
	cacheSignature       = 0x44495243
	cacheVersion         = 1
	dbEnvironment        = "SHA1_FILE_DIRECTORY"
	defaultDBEnvironment = ".dircache/objects"
)

type cacheHeader struct {
	signature uint32
	version   uint32
	entries   uint32
	sha1      [20]byte
}

type cacheTime struct {
	sec  uint32
	nsec uint32
}

type cacheEntry struct {
	ctime   cacheTime
	mtime   cacheTime
	stDev   uint32
	stIno   uint32
	stMode  uint32
	stUID   uint32
	stGID   uint32
	stSize  uint32
	sha1    [20]byte
	namelen uint16
	name    string
}

var (
	sha1FileDirectory string
	activeCache       []*cacheEntry
)

func cacheEntrySize(nameLen int) int {
	const base = 62
	return (base + nameLen + 8) &^ 7
}
