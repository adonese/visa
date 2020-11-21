FROM golang:latest

COPY . /app
WORKDIR /app
RUN go build
EXPOSE 8090
CMD ["/app/visa"]
