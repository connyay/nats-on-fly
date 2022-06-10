FROM nats:2.7.4-alpine3.15

ADD entrypoint.sh /fly/
ADD nats.conf /etc/nats.conf

ENTRYPOINT [ "fly/entrypoint.sh"]
CMD ["-c", "/etc/nats.conf"]
