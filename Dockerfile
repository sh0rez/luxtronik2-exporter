FROM golang:1.11-alpine as builder

ENV pkg="github.com/sh0rez/luxtronik2-exporter"
COPY . /go/src/$pkg
RUN printf "http://nl.alpinelinux.org/alpine/v3.8/main\nhttp://nl.alpinelinux.org/alpine/v3.8/community"  > /etc/apk/repositories
RUN apk add build-base &&\
  cd /go/src/$pkg &&\
  go build -ldflags '-s -w -extldflags "-static"' -a -o /luxtronik2-exporter . &&\
  ldd /luxtronik2-exporter

FROM alpine
COPY --from=builder /luxtronik2-exporter /luxtronik2-exporter
RUN chmod +x /luxtronik2-exporter
WORKDIR /lux
VOLUME /lux
CMD /luxtronik2-exporter
