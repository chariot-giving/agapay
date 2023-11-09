FROM golang:1.19-buster as build

WORKDIR /app

# Development build stage
# Air supports live reloading - https://github.com/cosmtrek/air
FROM build as development

RUN apt-get update && apt-get upgrade -y && \
    apt-get install -y git \
    make openssh-client

RUN go install github.com/cosmtrek/air@latest

RUN go install github.com/go-delve/delve/cmd/dlv@latest

# Port for the application
EXPOSE 8088
# Debugging port
EXPOSE 1357

ENTRYPOINT ["air"]

# Production build stage
FROM build as production-build-stage

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o agapay .

# Production
FROM gcr.io/distroless/base-debian11:latest as production

WORKDIR /

COPY --from=production-build-stage /app/agapay .

USER nonroot:nonroot

ENTRYPOINT ["./agapay"]
