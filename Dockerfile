FROM golang:1.22.1-alpine AS build 
ENV GO111MODULE on
ENV CGO_ENABLED 0

RUN apk add git make openssl

WORKDIR /go/src/github.com/karthick-kk/k8s-mutate-webhook-addca
ADD . .
# RUN make test
RUN make app

FROM scratch
WORKDIR /app
COPY --from=build /go/src/github.com/karthick-kk/k8s-mutate-webhook-addca/mutateme .
COPY --from=build /go/src/github.com/karthick-kk/k8s-mutate-webhook-addca/ssl ssl
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
CMD ["/app/mutateme"]
