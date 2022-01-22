FROM ubuntu:18.04
#Setup libimobile device, usbmuxd and some tools 
RUN export DEBIAN_FRONTEND=noninteractive && apt-get update && apt-get -y install unzip  wget curl libimobiledevice-utils libimobiledevice6 usbmuxd cmake git build-essential python

RUN apt update && apt install -y ffmpeg

#Grab go-ios from github and extract it in a folder
RUN wget https://github.com/danielpaulus/go-ios/releases/latest/download/go-ios-linux.zip
RUN mkdir go-ios
RUN unzip go-ios-linux.zip -d go-ios

#Grab GADS-docker-cli from github and extract it in /usr/local/bin
RUN wget https://github.com/shamanec/GADS-docker-cli/releases/latest/download/docker-cli.zip
RUN unzip docker-cli.zip -d /usr/local/bin

#Setup nvm and install latest appium
RUN curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.35.3/install.sh | bash
RUN export NVM_DIR="$HOME/.nvm" && [ -s "$NVM_DIR/nvm.sh" ] && \
     . "$NVM_DIR/nvm.sh" && nvm install 12.22.3 && \
    nvm alias default 12.22.3 && \
    npm config set user 0 && npm config set unsafe-perm true && npm install -g appium

#Copy scripts and WDA ipa to the image
COPY configs/nodeconfiggen.sh /opt/nodeconfiggen.sh
COPY configs/wda-sync.sh /
COPY ipa/WebDriverAgent.ipa /opt
COPY configs/supervision.p12 /opt
ENTRYPOINT ["/bin/bash","-c","/wda-sync.sh"]