FROM golang:1.17 AS build

ADD . /app
WORKDIR /app
RUN go build ./main.go

FROM ubuntu:20.04
EXPOSE 8080

WORKDIR /usr/src/app
COPY . .
COPY --from=build /app/main/ .
CMD ./main