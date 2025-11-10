FROM golang:1.22-bookworm

RUN apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends snmpd && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN install -Dm644 snmpd.conf /etc/snmp/snmpd.conf && \
    chmod +x docker/run-tests.sh

ENTRYPOINT ["/workspace/docker/run-tests.sh"]
CMD ["./..."]
