FROM golang:1.15-alpine as build-stage
MAINTAINER Daniel Carbone <daniel.p.carbone@gmail.com>
LABEL application=go-heicker
LABEL description="go-heicker build container"

RUN apk add --upgrade --no-cache g++ musl-dev git

COPY . /build
WORKDIR /build

RUN ./build.sh

FROM alpine:3
MAINTAINER Daniel Carbone <daniel.p.carbone@gmail.com>
LABEL application=go-heicker
LABEL description="go-heicker deploy container"

RUN apk add --upgrade --no-cache libstdc++ libgcc

WORKDIR /opt/go-heicker
COPY --from=build-stage /build/heicker ./
COPY public /opt/go-heicker/public

USER nobody

ENTRYPOINT [ "/opt/go-heicker/heicker" ]