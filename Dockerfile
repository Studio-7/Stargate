# FROM golang:1.12-alpine

# RUN apk add libvpx-dev
# RUN apk add screen
# RUN apk add git

# RUN go get github.com/sacOO7/gowebsocket
# RUN go get github.com/gorilla/websocket
# RUN go get github.com/pion/webrtc
# RUN go get github.com/kbinani/screenshot
# RUN go get github.com/go-vgo/robotgo
# RUN go get github.com/joho/godotenv
# RUN go get github.com/nfnt/resize

FROM ubuntu:18.04
WORKDIR /go/src/app

RUN apt-get update
RUN apt-get install libvpx-dev -y
RUN apt-get install screen -y
RUN apt-get install libpng-dev -y
RUN apt-get install gcc libc6-dev -y
RUN apt-get install libx11-dev xorg-dev libxtst-dev libpng++-dev -y

RUN apt-get install xcb libxcb-xkb-dev x11-xkb-utils libx11-xcb-dev libxkbcommon-x11-dev -y
RUN apt-get install libxkbcommon-dev -y

RUN apt-get install xsel xclip -y
ARG DEBIAN_FRONTEND=noninteractive
RUN apt-get install xserver-xorg-video-dummy -y
# COPY ./poi5305 /go/src/github.com/poi5305
RUN apt-get install libsdl-sound1.2-dev libsdl-image1.2-dev libsdl-gfx1.2-dev libsdl-console-dev libsdl1.2-dev -y
RUN apt-get install firefox -y
RUN apt-get install tmux -y

RUN apt-get install -y mesa-utils xserver-xorg-video-all mame alsa-base alsa-utils -y
RUN apt-get install libasound2 -y
COPY asound.conf /etc/asound.conf
# COPY default.conf /etc/modprobe.d/default.conf
# RUN useradd -ms /bin/bash gamer
# USER gamer
# WORKDIR /home/gamer
RUN apt-get install dosbox -y

COPY . .
ENV DISPLAY=:80
ENV SIGNAL="ws://127.0.0.1:9000/server"
# RUN chmod a+x ./run.sh
#RUN sh run.sh
#RUN screen -d -m X :2 -config dummy.conf
#ENV DISPLAY=:2
# RUN echo $DISPLAY
#RUN firefox


# ENV GOPROXY=direct
# ENV GO111MODULE=on
# ENV GOSUMDB=off
# RUN echo $GO111MODULE
# RUN go build -o main
ENTRYPOINT ["sh", "run.sh"]
