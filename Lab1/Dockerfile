# syntax=docker/dockerfile:1

# Name the stage `build`
FROM golang:1.27.3 AS build
WORKDIR /src

COPY go.mod *.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/http_server .

# Restart from empty file system and add the app/ directory.
# No go files etc.
FROM scratch
WORKDIR /app

# Copy the executable from the `build` stage
COPY --from=build /out/http_server /app/http_server
COPY --chown=10001:10001 public/ /app/public/

USER 10001:10001

EXPOSE 8080

ENTRYPOINT ["/app/http_server"]
CMD ["8080"]
