FROM golang:1.20-alpine as gobuild

WORKDIR /build
ADD . /build

# RUN go get -d -v ./...
RUN go env -w GO111MODULE=on
RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN go mod vendor
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o ./s3driver ./cmd/s3driver

FROM debian:buster-slim
LABEL maintainers="Ykk <445001186@qq.com>"
LABEL description="csi-s3 slim image"

# s3fs and some other dependencies
RUN apt-get update && \
  apt-get install -y \
  s3fs curl unzip && \
  rm -rf /var/lib/apt/lists/*

COPY --from=gobuild /build/s3driver /s3driver
ENTRYPOINT ["/s3driver"]
