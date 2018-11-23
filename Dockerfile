FROM golang:1.11-alpine as builder

ENV pkg="github.com/sh0rez/luxtronik2-exporter"
COPY . /go/src/$pkg
RUN echo "ipv6" >> /etc/modules &&\
  apk add build-base; \
  cd /go/src/$pkg &&\
  go build -ldflags '-s -w -extldflags "-static"' -a -o /luxtronik2-exporter . &&\
  ldd luxtronik2-exporter

FROM alpine
COPY --from=builder /luxtronik2-exporter /luxtronik2-exporter
RUN chmod +x /luxtronik2-exporter
CMD /luxtronik2-exporter
