// Copyright 2018 The agentx authors
// Licensed under the LGPLv3 with static-linking exception.
// See LICENCE file for details.

package agentx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
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
	if tcp, ok := conn.(*net.TCPConn); ok {
		_ = tcp.SetNoDelay(true)
		_ = tcp.SetKeepAlive(true)
	}
	c := &Client{
		logger:      options.logger,
		network:     network,
		address:     address,
		options:     options,
		conn:        conn,
		requestChan: make(chan *request, 64),
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
		ctx := context.Background()
		for headerPacket := range tx {
			headerPacketBytes, err := headerPacket.MarshalBinary()
			if err != nil {
				// recycle on error too
				if headerPacket.Header != nil {
					releaseHeader(headerPacket.Header)
				}
				releaseHeaderPacket(headerPacket)
				c.logger.Error("packet marshal error",
					getPacketHeaderSlogAttrs(headerPacket.Header),
					slog.Any("err", err),
				)
				continue
			}
			// Write all bytes to the connection (handle partial writes)
			left := headerPacketBytes
			for len(left) > 0 {
				n, err := c.conn.Write(left)
				if err != nil {
					if headerPacket.Header != nil {
						releaseHeader(headerPacket.Header)
					}
					releaseHeaderPacket(headerPacket)
					c.logger.Error("packet write error",
						getPacketHeaderSlogAttrs(headerPacket.Header),
						slog.Any("err", err),
					)
					break
				}
				left = left[n:]
			}
			if len(left) > 0 {
				// write failed; skip logging "sent"
				if headerPacket.Header != nil {
					releaseHeader(headerPacket.Header)
				}
				releaseHeaderPacket(headerPacket)
				c.logger.Error("packet write error",
					getPacketHeaderSlogAttrs(headerPacket.Header),
					slog.String("err", "short write"),
				)
				continue
			}
			if c.logger.Enabled(ctx, slog.LevelDebug) {
				c.logger.Debug("packet sent", getPacketHeaderSlogAttrs(headerPacket.Header))
			}
			// recycle header and headerPacket after successful send
			if headerPacket.Header != nil {
				releaseHeader(headerPacket.Header)
			}
			releaseHeaderPacket(headerPacket)
		}
	}()

	return tx
}

func (c *Client) runReceiver() chan *pdu.HeaderPacket {
	rx := make(chan *pdu.HeaderPacket)

	go func() {
		ctx := context.Background()
	mainLoop:
		for {
			headerBytes := acquireHeaderBuf()
			if _, err := io.ReadFull(c.conn, headerBytes[:]); err != nil {
				releaseHeaderBuf(headerBytes)
				if errors.Is(err, net.ErrClosed) {
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
						if tcp, ok := conn.(*net.TCPConn); ok {
							_ = tcp.SetNoDelay(true)
							_ = tcp.SetKeepAlive(true)
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
									return // from goroutine
								}
								c.sessions[session.ID()] = session
							}
							c.logger.Info("re-connect successful")
						}()
						continue mainLoop
					}
				}
				c.logger.Error("unexpected error", slog.Any("err", err))
				continue mainLoop
			}

			header := &pdu.Header{}
			if err := header.UnmarshalBinary(headerBytes[:]); err != nil {
				releaseHeaderBuf(headerBytes)
				c.logger.Error("header unmarshal error",
					getPacketHeaderSlogAttrs(header),
					slog.Any("err", err),
				)
				continue mainLoop
			}

			if c.logger.Enabled(ctx, slog.LevelDebug) {
				c.logger.Debug("packet received", getPacketHeaderSlogAttrs(header))
			}
			releaseHeaderBuf(headerBytes)

			var packet pdu.Packet
			switch header.Type {
			case pdu.TypeResponse:
				packet = &pdu.Response{}
			case pdu.TypeGet:
				packet = &pdu.Get{}
			case pdu.TypeGetNext:
				packet = &pdu.GetNext{}
			default:
				c.logger.Error("unable to handle packet", getPacketHeaderSlogAttrs(header))
				continue mainLoop
			}

			packetHandle, packetBytes := acquireIOBuf(int(header.PayloadLength))
			if _, err := io.ReadFull(c.conn, packetBytes); err != nil {
				releaseIOBuf(packetHandle)
				c.logger.Error("unable to read packet",
					getPacketHeaderSlogAttrs(header),
					slog.Any("err", err),
				)
				continue mainLoop
			}

			if err := packet.UnmarshalBinary(packetBytes); err != nil {
				releaseIOBuf(packetHandle)
				c.logger.Error("unable to unmarshal packet",
					getPacketHeaderSlogAttrs(header),
					slog.Any("err", err),
				)
				continue mainLoop
			}

			releaseIOBuf(packetHandle)
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
	req := acquireRequest()
	req.headerPacket = hp
	req.responseChan = make(chan *pdu.HeaderPacket, 1)
	c.requestChan <- req
	headerPacket := <-req.responseChan
	// recycle request struct
	req.headerPacket = nil
	req.responseChan = nil
	releaseRequest(req)
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
