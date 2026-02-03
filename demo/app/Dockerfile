FROM golang:1.22-alpine AS build
WORKDIR /src
COPY . .
RUN go build -o /bin/demo-app ./demo/app

FROM alpine:3.19
COPY --from=build /bin/demo-app /bin/demo-app
EXPOSE 8080
ENTRYPOINT ["/bin/demo-app"]
