FROM golang:1.21-alpine AS build
WORKDIR /app
COPY . ./
RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o bin/domino

FROM alpine AS release
COPY --from=build /app/bin/domino /usr/local/bin/domino
EXPOSE 8000
RUN addgroup --system nonroot && adduser --system nonroot --ingroup nonroot
USER nonroot:nonroot
ENTRYPOINT [ "domino" ]
