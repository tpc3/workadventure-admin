FROM golang AS build
WORKDIR /go/src/workadventure-admin

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN go build .

FROM debian:stable-slim
RUN apt-get update && \
    apt-get install -y ca-certificates && \
    rm -rf /var/lib/apt/lists/*
COPY --from=build /go/src/workadventure-admin/workadventure-admin /usr/local/bin
WORKDIR /data
ENTRYPOINT [ "/usr/local/bin/workadventure-admin" ]
