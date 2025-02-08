FROM docker.io/golang:alpine AS build

RUN apk update && apk add --no-cache git tzdata
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -o neurospecation ./

FROM alpine
RUN apk update && apk add --no-cache git # Need git for PR review diffs
COPY --from=build /build/neurospecation /neurospecation
ENTRYPOINT ["/neurospecation"]
CMD ["-h"]