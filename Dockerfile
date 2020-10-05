# build stage
FROM golang:1.14.8-alpine3.11 AS builder

LABEL stage=gggcp-intermediate

ENV GO111MODULE=on

ADD ./ /go/src/gggcp

RUN cd /go/src/gggcp && go build -mod vendor

FROM alpine:3.11.6

RUN apk add --no-cache tzdata

COPY --from=builder /go/src/gggcp/gggcp ./

