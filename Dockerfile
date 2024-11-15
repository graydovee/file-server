FROM golang:1.21 as builder

WORKDIR /source
ENV CGO_ENABLED=0

COPY go.mod go.mod
COPY go.sum go.sum

RUN go env -w GOPROXY='https://goproxy.cn,direct'
RUN go mod download

COPY cmd/ cmd/
COPY pkg/ pkg/


RUN CGO_ENABLED=0 go build -o /fileserver cmd/main.go

FROM alpine:3.16
WORKDIR /bin
COPY --from=builder /fileserver .
COPY template ./template
COPY assert ./assert
ENTRYPOINT ["/bin/fileserver"]
