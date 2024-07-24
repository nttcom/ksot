FROM golang:1.21.0-alpine

RUN apk update && apk add git && apk add alpine-sdk && apk add pcre2-dev && apk add cmake

WORKDIR /work/nb-server
COPY ./pkg ./pkg
COPY ./main.go ./
COPY ./go.mod ./
COPY ./go.sum ./
COPY ./libyang ./

WORKDIR /work/nb-server/libyang
RUN mkdir build; cd build
RUN cmake ..
RUN make
RUN make install

WORKDIR /work/nb-server
RUN go fmt ./...
RUN go vet ./...
RUN go test ./...
RUN go mod tidy
RUN go build -a -o ./app main.go

ENTRYPOINT ["./app"]