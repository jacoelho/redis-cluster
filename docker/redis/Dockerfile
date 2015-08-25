FROM gliderlabs/alpine

RUN apk --update add redis

COPY cluster.conf /cluster.conf

ENTRYPOINT ["redis-server", "/cluster.conf"]
