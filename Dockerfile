FROM namely/protoc as PROTOCOMPILER
WORKDIR /compile

COPY grpc/ .
RUN protoc --go-grpc_out=./ --go_out=./ ./*.proto && echo "built go protos"

FROM golang:alpine as BUILDER
WORKDIR /build

COPY Server/ ./
COPY --from=PROTOCOMPILER /compile/proto proto/

RUN CGO_ENABLED=0 GOOS=linux go build -o server

FROM scratch
#busybox
WORKDIR /app

COPY --from=BUILDER /build/server ./

EXPOSE 50051
CMD ["/app/server"]