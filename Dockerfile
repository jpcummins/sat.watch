FROM golang:1.23

ARG APP_VERSION

WORKDIR /app

COPY . ./

RUN CGO_ENABLED=0 GOOS=linux go build -o /satwatch

RUN CGO_ENABLED=0 GOOS=linux go build -o /create-user ./cmd/user/create

ENV APP_VERSION=${APP_VERSION}

EXPOSE 8080

CMD ["/satwatch"]
