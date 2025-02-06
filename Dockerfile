FROM docker.io/golang:alpine AS build

RUN apk update && apk add --no-cache git tzdata
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -o neurospecation ./cmd/neurospecation

FROM scratch
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /build/neurospecation /neurospecation
ENTRYPOINT [ "/neurospecation" ]
CMD []