FROM golang:1.23.6-alpine3.21 as builder

ENV BASE_APP_DIR /go/src/github.com/kyma-project/kyma-metrics-collector
WORKDIR ${BASE_APP_DIR}

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -v -o kyma-metrics-collector ./cmd/main.go
RUN mkdir /app && mv ./kyma-metrics-collector /app/kyma-metrics-collector

FROM gcr.io/distroless/static:nonroot
LABEL org.opencontainers.image.source="https://github.com/kyma-project/kyma-metrics-collector"

WORKDIR /app

COPY --from=builder /app /app
USER nonroot:nonroot

ENTRYPOINT ["/app/kyma-metrics-collector"]
