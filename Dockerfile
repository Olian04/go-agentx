FROM golang:1.22-alpine

RUN apk add --no-cache net-snmp

WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN install -Dm644 snmpd.conf /etc/snmp/snmpd.conf

CMD ["go", "test", "./..."]
