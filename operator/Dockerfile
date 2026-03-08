FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /operator .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /operator /operator
ENTRYPOINT ["/operator"]
