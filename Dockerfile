# Building the binary of the App
FROM golang:latest AS build


WORKDIR /go/src/bot

# Copy all the Code and stuff to compile everything
COPY . .

RUN go mod tidy


RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -X 'main.Version=1.0.2' -X 'main.BuildTime=$(date +%s)'" -a -installsuffix cgo -o main .


FROM alpine:latest as release

WORKDIR /app

COPY --from=build /go/src/bot/main .
COPY --from=build /go/src/bot/.env .
COPY --from=build /go/src/bot/mediamtx.yml .
COPY --from=build /go/src/bot/dist ./dist


RUN apk -U upgrade \
    && apk add --no-cache ca-certificates tzdata curl\
    && chmod +x /app/main 
ENV PATH="/app:${PATH}"

ENV TZ=Asia/Ulaanbaatar
RUN curl -L https://github.com/bluenviron/mediamtx/releases/download/v1.11.3/mediamtx_v1.11.3_linux_amd64.tar.gz| tar xz -C /app
COPY mediamtx.yml /app/mediamtx.yml
RUN chmod +x /app/mediamtx  # Ensure MediaMTX is executable

EXPOSE 3000 8554 8000 8889

ENTRYPOINT ["./main"]