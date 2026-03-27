# C version (initial Git) rules are kept for reference.
# CFLAGS=-g
# CC=gcc
# PROG=update-cache show-diff init-db write-tree read-tree commit-tree cat-file
# all: $(PROG)
# install: $(PROG)
# 	install $(PROG) $(HOME)/bin/
# LIBS= -lssl
# init-db: init-db.o
# update-cache: update-cache.o read-cache.o
# 	$(CC) $(CFLAGS) -o update-cache update-cache.o read-cache.o $(LIBS)
# show-diff: show-diff.o read-cache.o
# 	$(CC) $(CFLAGS) -o show-diff show-diff.o read-cache.o $(LIBS)
# write-tree: write-tree.o read-cache.o
# 	$(CC) $(CFLAGS) -o write-tree write-tree.o read-cache.o $(LIBS)
# read-tree: read-tree.o read-cache.o
# 	$(CC) $(CFLAGS) -o read-tree read-tree.o read-cache.o $(LIBS)
# commit-tree: commit-tree.o read-cache.o
# 	$(CC) $(CFLAGS) -o commit-tree commit-tree.o read-cache.o $(LIBS)
# cat-file: cat-file.o read-cache.o
# 	$(CC) $(CFLAGS) -o cat-file cat-file.o read-cache.o $(LIBS)
# read-cache.o: cache.h
# show-diff.o: cache.h
# clean:
# 	rm -f *.o $(PROG) temp_git_file_*
# backup: clean
# 	cd .. ; tar czvf dircache.tar.gz dir-cache

GO ?= go
export GOCACHE ?= $(CURDIR)/.cache/go-build
DESTDIR ?= $(HOME)/bin

PROG = update-cache show-diff init-db write-tree read-tree commit-tree cat-file
COMMON = cache.go read-cache.go

all: $(PROG)

init-db: init-db.go $(COMMON)
	$(GO) build -tags init_db -o $@ init-db.go $(COMMON)

update-cache: update-cache.go $(COMMON)
	$(GO) build -tags update_cache -o $@ update-cache.go $(COMMON)

show-diff: show-diff.go $(COMMON)
	$(GO) build -tags show_diff -o $@ show-diff.go $(COMMON)

write-tree: write-tree.go $(COMMON)
	$(GO) build -tags write_tree -o $@ write-tree.go $(COMMON)

read-tree: read-tree.go $(COMMON)
	$(GO) build -tags read_tree -o $@ read-tree.go $(COMMON)

commit-tree: commit-tree.go $(COMMON)
	$(GO) build -tags commit_tree -o $@ commit-tree.go $(COMMON)

cat-file: cat-file.go $(COMMON)
	$(GO) build -tags cat_file -o $@ cat-file.go $(COMMON)

install: $(PROG)
	install -m 0755 $(PROG) $(DESTDIR)/

clean:
	rm -f $(PROG) temp_git_file_*
	rm -rf .cache/go-build
