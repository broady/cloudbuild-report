FROM alpine
RUN apk add --update ca-certificates
COPY gopath/bin/report /report
ENTRYPOINT /report
