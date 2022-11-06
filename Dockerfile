
ARG GO_VERSION=1.14
FROM golang:${GO_VERSION}-alpine AS dev

RUN apk add --no-cache ca-certificates git

COPY .netrc /root/.netrc
RUN chmod 600 /root/.netrc

ENV APP_NAME="main" \
    APP_PATH="/var/app"

ENV APP_BUILD_NAME="${APP_NAME}"

COPY . ${APP_PATH}
WORKDIR ${APP_PATH}

ENV GO111MODULE="on" \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOFLAGS="-mod=vendor"

ENTRYPOINT ["sh"]

FROM dev as build

RUN (([ ! -d "${APP_PATH}/vendor" ] && go mod download && go mod vendor) || true)
RUN GIT_COMMIT=$(git rev-list -1 HEAD) && \ 
    BUILD=$(date +%FT%T%z) && \
    go build -ldflags="-s -w -X 'main.Version=${GIT_COMMIT}' -X main.Build=${BUILD}" -mod vendor -o ${APP_BUILD_NAME} main.go
RUN chmod +x ${APP_BUILD_NAME}

FROM riveriacontregistry.azurecr.io/primusquery-buster-slim:latest AS prod

ENV APP_BUILD_PATH="/var/app" \
    APP_BUILD_NAME="main"
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
WORKDIR /home/job
USER job
RUN mkdir log
COPY --from=build ${APP_BUILD_PATH}/${APP_BUILD_NAME} /home/job

#EXPOSE ${APP_PORT}
ENTRYPOINT ["/home/job/main"]
CMD ""