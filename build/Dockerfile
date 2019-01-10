FROM alpine:3.8

RUN apk upgrade --update --no-cache

USER nobody

ADD build/_output/bin/lyra-operator /usr/local/bin/lyra-operator
