FROM ubuntu:14.04.4

COPY bin/dmp /usr/bin/dmp

EXPOSE 7946 7373
EXPOSE 7946/udp 7373/udp
EXPOSE 8080

ENTRYPOINT ["/usr/bin/dmp"]
