// Copyright 2018 The agentx authors
// Licensed under the LGPLv3 with static-linking exception.
// See LICENCE file for details.

package agentx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Olian04/go-agentx/pdu"
	"github.com/Olian04/go-agentx/value"
)

// Session defines an agentx session.
type Session struct {
	client    *Client
	handler   Handler
	sessionID uint32
	timeout   time.Duration

	openRequestPacket     *pdu.HeaderPacket
	registerRequestPacket *pdu.HeaderPacket
}

func openSession(client *Client, nameOID value.OID, name string, handler Handler) (*Session, error) {
	s := &Session{
		client:  client,
		handler: handler,
		timeout: client.options.timeout,
	}

	requestPacket := &pdu.Open{}
	requestPacket.Timeout.Duration = s.timeout
	requestPacket.ID.SetIdentifier(nameOID)
	requestPacket.Description.Text = name
	request := &pdu.HeaderPacket{Header: &pdu.Header{Type: pdu.TypeOpen}, Packet: requestPacket}

	response := s.request(request)
	if err := checkError(response); err != nil {
		return nil, err
	}
	s.sessionID = response.Header.SessionID
	s.openRequestPacket = request

	return s, nil
}

// ID returns the session id.
func (s *Session) ID() uint32 {
	return s.sessionID
}

// Register registers the client under the provided rootID with the provided priority
// on the master agent.
func (s *Session) Register(priority byte, baseOID value.OID) error {
	if s.registerRequestPacket != nil {
		return fmt.Errorf("session is already registered")
	}

	requestPacket := &pdu.Register{}
	requestPacket.Timeout.Duration = s.timeout
	requestPacket.Timeout.Priority = priority
	requestPacket.Subtree.SetIdentifier(baseOID)
	request := &pdu.HeaderPacket{Header: &pdu.Header{Type: pdu.TypeRegister}, Packet: requestPacket}

	response := s.request(request)
	if err := checkError(response); err != nil {
		return err
	}
	s.registerRequestPacket = request
	return nil
}

// Unregister removes the registration for the provided subtree.
func (s *Session) Unregister(priority byte, baseOID value.OID) error {
	if s.registerRequestPacket == nil {
		return fmt.Errorf("session is not registered")
	}

	requestPacket := &pdu.Unregister{}
	requestPacket.Timeout.Duration = s.timeout
	requestPacket.Timeout.Priority = priority
	requestPacket.Subtree.SetIdentifier(baseOID)
	request := &pdu.HeaderPacket{Header: &pdu.Header{}, Packet: requestPacket}

	response := s.request(request)
	if err := checkError(response); err != nil {
		return err
	}
	s.registerRequestPacket = nil
	return nil
}

// Close tears down the session with the master agent.
func (s *Session) Close() error {
	requestPacket := &pdu.Close{Reason: pdu.ReasonShutdown}

	response := s.request(&pdu.HeaderPacket{Header: &pdu.Header{}, Packet: requestPacket})
	if err := checkError(response); err != nil {
		return err
	}
	return nil
}

func (s *Session) reopen() error {
	if s.openRequestPacket != nil {
		response := s.request(s.openRequestPacket)
		if err := checkError(response); err != nil {
			return err
		}
		s.sessionID = response.Header.SessionID
	}

	if s.registerRequestPacket != nil {
		response := s.request(s.registerRequestPacket)
		if err := checkError(response); err != nil {
			return err
		}
	}

	return nil
}

func (s *Session) request(hp *pdu.HeaderPacket) *pdu.HeaderPacket {
	hp.Header.SessionID = s.sessionID
	return s.client.request(hp)
}

func (s *Session) handle(request *pdu.HeaderPacket) *pdu.HeaderPacket {
	responseHeader := &pdu.Header{}
	responseHeader.SessionID = request.Header.SessionID
	responseHeader.TransactionID = request.Header.TransactionID
	responseHeader.PacketID = request.Header.PacketID
	// We always encode using little-endian (FlagNetworkByteOrder unset)
	// Ensure the response header Flags reflect our encoding.
	responseHeader.Flags = 0
	responsePacket := &pdu.Response{}

	ctx := context.Background()
	ctx = withSessionID(ctx, request.Header.SessionID)
	ctx = withTransactionID(ctx, request.Header.TransactionID)
	ctx = withPacketID(ctx, request.Header.PacketID)

	switch requestPacket := request.Packet.(type) {
	case *pdu.Get:
		if s.handler == nil {
			s.client.logger.Warn("no handler for session specified")
			// Return Null for each requested OID
			for _, sr := range requestPacket.SearchRanges {
				responsePacket.Variables.Add(sr.From.GetIdentifier(), pdu.VariableTypeNull, nil)
			}
			break
		}

		// One response varbind per requested OID
		for _, sr := range requestPacket.SearchRanges {
			reqOID := sr.From.GetIdentifier()
			oid, t, v, err := s.handler.Get(ctx, reqOID)
			if err != nil {
				s.client.logger.Error("packet error", slog.Any("err", err))
				responsePacket.Error = pdu.ErrorProcessing
			}
			if oid == nil {
				responsePacket.Variables.Add(reqOID, pdu.VariableTypeNoSuchObject, nil)
			} else {
				responsePacket.Variables.Add(oid, t, v)
			}
		}

	case *pdu.GetNext:
		if s.handler == nil {
			s.client.logger.Warn("no handler for session specified")
			break
		}

		for _, sr := range requestPacket.SearchRanges {
			oid, t, v, err := s.handler.GetNext(ctx, sr.From.GetIdentifier(), (sr.From.Include == 1), sr.To.GetIdentifier())
			if err != nil {
				s.client.logger.Error("packet error", slog.Any("err", err))
				responsePacket.Error = pdu.ErrorProcessing
			}

			if oid == nil {
				responsePacket.Variables.Add(sr.From.GetIdentifier(), pdu.VariableTypeEndOfMIBView, nil)
			} else {
				responsePacket.Variables.Add(oid, t, v)
			}
		}

	case *pdu.GetBulk:
		if s.handler == nil {
			s.client.logger.Warn("no handler for session specified")
			break
		}

		searchRanges := requestPacket.SearchRanges
		totalRanges := len(searchRanges)

		nonRepeaters := int(requestPacket.NonRepeaters)
		if nonRepeaters > totalRanges {
			nonRepeaters = totalRanges
		}

		for index := 0; index < nonRepeaters; index++ {
			sr := searchRanges[index]
			from := sr.From.GetIdentifier()
			include := sr.From.Include == 1
			to := sr.To.GetIdentifier()

			oid, t, v, err := s.handler.GetNext(ctx, from, include, to)
			if err != nil {
				s.client.logger.Error("packet error", slog.Any("err", err))
				responsePacket.Error = pdu.ErrorProcessing
			}

			if oid == nil {
				responsePacket.Variables.Add(from, pdu.VariableTypeEndOfMIBView, nil)
			} else {
				responsePacket.Variables.Add(oid, t, v)
			}
		}

		remaining := totalRanges - nonRepeaters
		if remaining <= 0 || requestPacket.MaxRepetitions == 0 {
			break
		}

		currentFrom := make([]value.OID, remaining)
		includeFrom := make([]bool, remaining)
		toOIDs := make([]value.OID, remaining)
		exhausted := make([]bool, remaining)

		for index := 0; index < remaining; index++ {
			sr := searchRanges[nonRepeaters+index]
			currentFrom[index] = sr.From.GetIdentifier()
			includeFrom[index] = sr.From.Include == 1
			toOIDs[index] = sr.To.GetIdentifier()
		}

		maxRepetitions := int(requestPacket.MaxRepetitions)
		for repetition := 0; repetition < maxRepetitions; repetition++ {
			for index := 0; index < remaining; index++ {
				if exhausted[index] {
					responsePacket.Variables.Add(currentFrom[index], pdu.VariableTypeEndOfMIBView, nil)
					continue
				}

				oid, t, v, err := s.handler.GetNext(ctx, currentFrom[index], includeFrom[index], toOIDs[index])
				if err != nil {
					s.client.logger.Error("packet error", slog.Any("err", err))
					responsePacket.Error = pdu.ErrorProcessing
				}

				if oid == nil {
					responsePacket.Variables.Add(currentFrom[index], pdu.VariableTypeEndOfMIBView, nil)
					exhausted[index] = true
					includeFrom[index] = false
					continue
				}

				responsePacket.Variables.Add(oid, t, v)
				currentFrom[index] = oid
				includeFrom[index] = false
			}
		}

	default:
		s.client.logger.Error("unable to handle packet", slog.String("packet-type", request.Header.Type.String()))
		responsePacket.Error = pdu.ErrorProcessing
	}

	return &pdu.HeaderPacket{Header: responseHeader, Packet: responsePacket}
}

func checkError(hp *pdu.HeaderPacket) error {
	response, ok := hp.Packet.(*pdu.Response)
	if !ok {
		return nil
	}
	if response.Error == pdu.ErrorNone {
		return nil
	}
	return errors.New(response.Error.String())
}
