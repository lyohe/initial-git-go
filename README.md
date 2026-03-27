# initial-git-go

Go re-implementation of the initial Git commit e83c5163316f89bfbde7d9ab23ca2e25604af290.

```
mkdir -p work
docker build -t initial-git-go .
docker run --rm -it -v "$PWD/work:/work" initial-git-go
```
