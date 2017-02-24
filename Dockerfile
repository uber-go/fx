FROM golang:1.8.0

WORKDIR /go/src/go.uber.org/fx
ADD .build/deps.mk /go/src/go.uber.org/fx/.build/
ADD glide.yaml glide.lock /go/src/go.uber.org/fx/
RUN make -f .build/deps.mk deps
ADD . /go/src/go.uber.org/fx/
RUN make clean
