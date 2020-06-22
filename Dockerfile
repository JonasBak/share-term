FROM golang:alpine as build

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . /build

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o share-term .

FROM scratch

COPY --from=build /build/share-term .

COPY ./templates ./templates

EXPOSE 8080

CMD ["./share-term", "-server", "-addr", "0.0.0.0:8080"]
