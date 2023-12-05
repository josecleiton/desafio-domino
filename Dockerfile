FROM golang:1.21-alpine AS build
WORKDIR /app
COPY . ./
RUN go mod download
RUN go build -ldflags "-s -w" -o bin/domino

FROM gcr.io/distroless/base-debian12 AS release
COPY --from=build /app/bin/domino /usr/local/bin/domino
EXPOSE 8000
USER nonroot:nonroot
ENTRYPOINT [ "domino" ]
