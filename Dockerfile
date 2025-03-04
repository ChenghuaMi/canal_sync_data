FROM golang:1.23 AS builder

COPY go.mod go.sum /orca_sync_service/
WORKDIR /orca_sync_service

RUN go env -w GOPROXY='https://goproxy.io,direct' \
    && go mod download

COPY pkg /orca_sync_service/pkg
COPY main.go /orca_sync_service/


RUN go build -o orca_sync_service main.go

FROM ubuntu:22.04 AS runner

COPY --from=builder /orca_sync_service/orca_sync_service /orca_sync_service/orca_sync_service
COPY etc /orca_sync_service/etc
WORKDIR /orca_sync_service
CMD ["/orca_sync_service/orca_sync_service" ]