FROM golang:1.20 AS build

WORKDIR /src
COPY . .

RUN make
RUN make install DESTDIR=/usr/local/bin

FROM debian:stable-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
	bsdextrautils \
	diffutils \
	&& rm -rf /var/lib/apt/lists/*

COPY --from=build /usr/local/bin/init-db /usr/local/bin/
COPY --from=build /usr/local/bin/update-cache /usr/local/bin/
COPY --from=build /usr/local/bin/write-tree /usr/local/bin/
COPY --from=build /usr/local/bin/read-tree /usr/local/bin/
COPY --from=build /usr/local/bin/commit-tree /usr/local/bin/
COPY --from=build /usr/local/bin/cat-file /usr/local/bin/
COPY --from=build /usr/local/bin/show-diff /usr/local/bin/

WORKDIR /work
CMD ["/bin/bash"]
