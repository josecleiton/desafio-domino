FROM golang:1.21-alpine AS build
WORKDIR /app
COPY . ./
RUN go mod download
RUN CGO_ENABLED=0 go build -o bin/domino

FROM gcr.io/distroless/base-debian11 AS release
COPY --from=build /app/bin/domino /usr/local/bin/domino
EXPOSE 8000
USER nonroot:nonroot
ENTRYPOINT [ "domino" ]
