# Building the binary of the App
FROM golang:latest AS build


WORKDIR /go/src/bot

# Copy all the Code and stuff to compile everything
COPY . .

RUN go mod tidy


RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -X 'main.Version=1.0.2' -X 'main.BuildTime=$(date +%s)'" -a -installsuffix cgo -o main .


FROM alpine:latest as release

WORKDIR /app


RUN mkdir ./messages 


COPY --from=build /go/src/bot/main .
COPY --from=build /go/src/bot/.env .
COPY --from=build /go/src/bot/*.json .

RUN apk -U upgrade \
    && apk add --no-cache ca-certificates tzdata curl\
    && chmod +x /app/main 
ENV TZ=Asia/Ulaanbaatar


EXPOSE 3000

ENTRYPOINT ["./main"]