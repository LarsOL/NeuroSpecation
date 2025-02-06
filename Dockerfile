FROM docker.io/golang AS build

RUN apk update && apk add --no-cache git tzdata
WORKDIR /build
COPY . .
RUN CGO_ENABLED=0 go build -ldflags -o neurospecation ./cmd/neurospecation

FROM scratch
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /build/neurospecation /neurospecation
CMD [ "/neurospecation" ]