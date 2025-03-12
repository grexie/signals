FROM golang:alpine AS build

WORKDIR /app

COPY . /app

RUN go build -o /signals . 

FROM alpine:latest

COPY --from=build /signals /signals

CMD ["/signals"]