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
RUN apt-get install libsdl-sound1.2-dev libsdl-image1.2-dev libsdl-gfx1.2-dev libsdl-console-dev libsdl1.2-dev -y
RUN apt-get install firefox -y
RUN apt-get install tmux -y

RUN apt-get install -y mesa-utils xserver-xorg-video-all mame alsa-base alsa-utils -y
RUN apt-get install libasound2 -y
RUN apt-get install dosbox -y

COPY . .
# ENV DISPLAY=:80
ENV SIGNAL="ws://127.0.0.1:9000/server"
ENTRYPOINT ["sh", "run.sh"]
