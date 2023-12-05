FROM golang:1.21-alpine as build
WORKDIR /app
COPY . ./
RUN go mod download
RUN go build -o bin/domino

FROM alpine
COPY --from=build /app/bin/domino /usr/local/bin/domino
EXPOSE 8080

ENTRYPOINT [ "domino" ]
