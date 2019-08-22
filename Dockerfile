FROM golang:1.8

WORKDIR /go/src/app

COPY . .
RUN apt-get update
#RUN apt-get upgrade -y
RUN apt-get install -y liblzma-dev
RUN go get -d -v ./...
RUN go install -v ./...

CMD gozimhttpd -path=$ZIM_PATH -index=$INDEX_PATH
