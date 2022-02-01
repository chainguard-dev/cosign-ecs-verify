FROM alpine:3.14
ARG signed
ENV SIGNED=$signed
CMD sleep 1000
