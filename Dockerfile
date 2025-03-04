FROM golang:1.24.1 AS builder

WORKDIR /go/src/github.com/kyma-project/kyma-metrics-collector

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -v -o kyma-metrics-collector ./cmd/main.go
RUN mkdir /app && mv ./kyma-metrics-collector /app/kyma-metrics-collector

FROM scratch
LABEL org.opencontainers.image.source="https://github.com/kyma-project/kyma-metrics-collector"

WORKDIR /app

COPY --from=builder /app /app
USER 65532:65532

ENTRYPOINT ["/app/kyma-metrics-collector"]
