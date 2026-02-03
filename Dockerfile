FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/klyr ./cmd/klyr

FROM alpine:3.19
COPY --from=build /bin/klyr /bin/klyr
ENTRYPOINT ["/bin/klyr"]
