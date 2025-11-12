// Copyright 2018 The agentx authors
// Licensed under the LGPLv3 with static-linking exception.
// See LICENCE file for details.

package agentx

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/Olian04/go-agentx/pdu"
	"github.com/Olian04/go-agentx/value"
)

// Client defines an agentx client.
type Client struct {
	logger      *slog.Logger
	network     string
	address     string
	options     dialOptions
	conn        net.Conn
	requestChan chan *request
	sessions    map[uint32]*Session
}

// Dial connects to the provided agentX endpoint.
func Dial(network, address string, opts ...DialOption) (*Client, error) {
	options := dialOptions{}
	for _, dialOption := range opts {
		dialOption(&options)
	}

	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, fmt.Errorf("dial %s %s: %w", network, address, err)
	}
	c := &Client{
		logger:      options.logger,
		network:     network,
		address:     address,
		options:     options,
		conn:        conn,
		requestChan: make(chan *request),
		sessions:    make(map[uint32]*Session),
	}

	if c.logger == nil {
		c.logger = slog.New(slog.DiscardHandler)
	}

	tx := c.runTransmitter()
	rx := c.runReceiver()
	c.runDispatcher(tx, rx)

	return c, nil
}

// Close tears down the client.
func (c *Client) Close() error {
	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("close connection: %w", err)
	}
	return nil
}

// Session sets up a new session.
func (c *Client) Session(nameOID value.OID, name string, handler Handler) (*Session, error) {
	s, err := openSession(c, nameOID, name, handler)
	if err != nil {
		return nil, fmt.Errorf("open session: %w", err)
	}
	c.sessions[s.ID()] = s
	return s, nil
}

func (c *Client) runTransmitter() chan *pdu.HeaderPacket {
	tx := make(chan *pdu.HeaderPacket)

	go func() {
		for headerPacket := range tx {
			headerPacketBytes, err := headerPacket.MarshalBinary()
			if err != nil {
				c.logger.Error("packet marshal error",
					getPacketHeaderSlogAttrs(headerPacket.Header),
					slog.Any("err", err),
				)
				continue
			}
			writer := bufio.NewWriter(c.conn)
			if _, err := writer.Write(headerPacketBytes); err != nil {
				c.logger.Error("packet write error",
					getPacketHeaderSlogAttrs(headerPacket.Header),
					slog.Any("err", err),
				)
				continue
			}
			if err := writer.Flush(); err != nil {
				c.logger.Error("packet flush error",
					getPacketHeaderSlogAttrs(headerPacket.Header),
					slog.Any("err", err),
				)
				continue
			}
			c.logger.Debug("packet sent",
				getPacketHeaderSlogAttrs(headerPacket.Header),
			)
		}
	}()

	return tx
}

func (c *Client) runReceiver() chan *pdu.HeaderPacket {
	rx := make(chan *pdu.HeaderPacket)

	go func() {
	mainLoop:
		for {
			reader := bufio.NewReader(c.conn)
			headerBytes := make([]byte, pdu.HeaderSize)
			if _, err := reader.Read(headerBytes); err != nil {
				if opErr, ok := err.(*net.OpError); ok && strings.HasSuffix(opErr.Error(), "use of closed network connection") {
					return
				}
				if err == io.EOF {
					c.logger.Info("lost connection", slog.Duration("re-connect-in", c.options.reconnectInterval))
				reopenLoop:
					for {
						time.Sleep(c.options.reconnectInterval)
						conn, err := net.Dial(c.network, c.address)
						if err != nil {
							c.logger.Error("re-connect error", slog.Any("err", err))
							continue reopenLoop
						}
						c.conn = conn
						go func() {
							for _, session := range c.sessions {
								delete(c.sessions, session.ID())
								if err := session.reopen(); err != nil {
									c.logger.Error("re-open error",
										getPacketHeaderSlogAttrs(session.openRequestPacket.Header),
										slog.Any("err", err),
									)
									return
								}
								c.sessions[session.ID()] = session
							}
							c.logger.Info("re-connect successful")
						}()
						continue mainLoop
					}
				}
				panic(err)
			}

			header := &pdu.Header{}
			if err := header.UnmarshalBinary(headerBytes); err != nil {
				c.logger.Error("header unmarshal error",
					getPacketHeaderSlogAttrs(header),
					slog.Any("err", err),
				)
				continue mainLoop
			}

			c.logger.Debug("packet received", getPacketHeaderSlogAttrs(header))

			var packet pdu.Packet
			switch header.Type {
			case pdu.TypeResponse:
				packet = &pdu.Response{}
			case pdu.TypeGet:
				packet = &pdu.Get{}
			case pdu.TypeGetNext:
				packet = &pdu.GetNext{}
			case pdu.TypeGetBulk:
				packet = &pdu.GetBulk{}
			default:
				c.logger.Error("unable to handle packet", getPacketHeaderSlogAttrs(header))
				continue mainLoop
			}

			packetBytes := make([]byte, header.PayloadLength)
			if _, err := reader.Read(packetBytes); err != nil {
				c.logger.Error("unable to read packet",
					getPacketHeaderSlogAttrs(header),
					slog.Any("err", err),
				)
				continue mainLoop
			}

			if err := packet.UnmarshalBinary(packetBytes); err != nil {
				c.logger.Error("unable to unmarshal packet",
					getPacketHeaderSlogAttrs(header),
					slog.Any("err", err),
				)
				continue mainLoop
			}

			rx <- &pdu.HeaderPacket{Header: header, Packet: packet}
		}
	}()

	return rx
}

func (c *Client) runDispatcher(tx, rx chan *pdu.HeaderPacket) {
	go func() {
		currentPacketID := uint32(0)
		responseChans := make(map[uint32]chan *pdu.HeaderPacket)

		for {
			select {
			case request := <-c.requestChan:
				request.headerPacket.Header.PacketID = currentPacketID
				responseChans[currentPacketID] = request.responseChan
				currentPacketID++
				tx <- request.headerPacket

			case headerPacket := <-rx:
				if responseChan, ok := responseChans[headerPacket.Header.PacketID]; ok {
					responseChan <- headerPacket
					delete(responseChans, headerPacket.Header.PacketID)
				} else if session, ok := c.sessions[headerPacket.Header.SessionID]; ok {
					tx <- session.handle(headerPacket)
				} else {
					c.logger.Error("got packet without session",
						getPacketHeaderSlogAttrs(headerPacket.Header),
						slog.Int("awaiting_responses", len(responseChans)),
					)
				}
			}
		}
	}()
}

func (c *Client) request(hp *pdu.HeaderPacket) *pdu.HeaderPacket {
	responseChan := make(chan *pdu.HeaderPacket)
	request := &request{
		headerPacket: hp,
		responseChan: responseChan,
	}
	c.requestChan <- request
	headerPacket := <-responseChan
	return headerPacket
}

func getPacketHeaderSlogAttrs(header *pdu.Header) slog.Attr {
	return slog.GroupAttrs("packet_header",
		slog.String("packet_type", header.Type.String()),
		slog.Any("session_id", header.SessionID),
		slog.Any("transaction_id", header.TransactionID),
		slog.Any("packet_id", header.PacketID),
		slog.Any("payload_length", header.PayloadLength),
	)
}
