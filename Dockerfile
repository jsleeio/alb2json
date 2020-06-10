FROM scratch
COPY alb2json /alb2json
USER 1000
ENTRYPOINT ["/alb2json"]

