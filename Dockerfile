FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/server ./cmd/server

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/server /server
ENV DATA_FILE=/data/links.json
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["/server"]
