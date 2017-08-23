FROM alpine:latest
COPY ./sts /sts
RUN apk update && apk add libc6-compat

EXPOSE 8080

ENTRYPOINT [ "/sts" ]
