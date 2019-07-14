FROM golang:1.12 as builder
WORKDIR /build
COPY . /build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o bot .

FROM gcr.io/distroless/base
WORKDIR /bot
COPY --from=builder /build/bot .
CMD ["/bot/bot"]

