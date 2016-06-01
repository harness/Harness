FROM catalog.shurenyun.com/library/drone_build:0.1

COPY . /go/src/github.com/drone_build
WORKDIR /go/src/github.com/drone_build

ENV PATH $PATH:/go/bin
ENV GO15VENDOREXPERIMENT 1

RUN make gen_static && make build_static

ADD .env.sample .env

ENTRYPOINT ["./drone_static"]

EXPOSE 9898


