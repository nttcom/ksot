FROM golang:1.21.0-alpine
ARG GITHUB_REPO_URL
ARG GITHUB_USER_NAME
ARG GITHUB_USER_MAIL


RUN apk update && apk add git
RUN mkdir -p /work/github-server/gitrepo

WORKDIR /work/github-server/gitrepo
RUN echo ${GITHUB_REPO_URL}
RUN git clone ${GITHUB_REPO_URL}
RUN git config --global user.name ${GITHUB_USER_NAME}
RUN git config --global user.email ${GITHUB_USER_MAIL}

WORKDIR /work/github-server
COPY ./pkg ./pkg
COPY ./main.go ./
COPY ./go.mod ./
COPY ./go.sum ./

RUN go mod tidy
RUN go build -a -o ./app main.go
ENTRYPOINT ["./app"]