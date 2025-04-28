FROM --platform=$BUILDPLATFORM golang:1.23 AS builder

ARG APP_VERSION
ARG TARGETPLATFORM
ARG BUILDPLATFORM

WORKDIR /app

COPY . ./

# Build for the target platform
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$(echo $TARGETPLATFORM | cut -d/ -f2) go build -o /satwatch
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$(echo $TARGETPLATFORM | cut -d/ -f2) go build -o /create-user ./cmd/user/create

# Use a minimal base image for the final stage
FROM --platform=$TARGETPLATFORM alpine:latest

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /satwatch /app/satwatch
COPY --from=builder /create-user /app/create-user

ENV APP_VERSION=${APP_VERSION}

EXPOSE 8080

CMD ["/app/satwatch"]
