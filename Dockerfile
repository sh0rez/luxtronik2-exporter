FROM golang:1.13 as builder
COPY . /build
RUN cd /build && make static

FROM alpine
COPY --from=builder /build/luxtronik2-exporter /usr/local/bin/luxtronik2-exporter
WORKDIR /lux
ENTRYPOINT ["/usr/local/bin/luxtronik2-exporter"]
