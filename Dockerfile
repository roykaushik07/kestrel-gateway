FROM golang:1.23 AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/gateway ./cmd/gateway

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/gateway /gateway
COPY config/routes.yaml /config/routes.yaml
ENV ROUTES_FILE=/config/routes.yaml
EXPOSE 8080
ENTRYPOINT ["/gateway"]
