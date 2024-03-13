FROM golang:1.21-alpine as builder
WORKDIR /app

COPY . .

RUN go build -o /proxy

FROM alpine
RUN apk --no-cache add ca-certificates
COPY --from=builder /proxy /proxy
WORKDIR /
CMD ["/proxy"]
