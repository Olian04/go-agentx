// Copyright 2018 The agentx authors
// Licensed under the LGPLv3 with static-linking exception.
// See LICENCE file for details.
//
// Pooling utilities for buffers and request structs.

package agentx

import (
	"sync"

	"github.com/Olian04/go-agentx/pdu"
)

const pooledBufCap = 8 << 10 // 8KB default pooled buffer size

var (
	headerBufPool = sync.Pool{
		New: func() any {
			var b [pdu.HeaderSize]byte
			return &b
		},
	}
	ioBufPool = sync.Pool{
		New: func() any { return &pooledBytes{b: make([]byte, pooledBufCap)} },
	}
	requestPool = sync.Pool{
		New: func() any { return &request{} },
	}
	headerPool = sync.Pool{
		New: func() any { return &pdu.Header{} },
	}
	headerPacketPool = sync.Pool{
		New: func() any { return &pdu.HeaderPacket{} },
	}
)

type pooledBytes struct {
	b []byte
}

func acquireHeaderBuf() *[pdu.HeaderSize]byte {
	return headerBufPool.Get().(*[pdu.HeaderSize]byte)
}

func releaseHeaderBuf(b *[pdu.HeaderSize]byte) {
	headerBufPool.Put(b)
}

func acquireIOBuf(n int) (*pooledBytes, []byte) {
	p := ioBufPool.Get().(*pooledBytes)
	if cap(p.b) < n {
		p.b = make([]byte, n)
	}
	return p, p.b[:n]
}

func releaseIOBuf(p *pooledBytes) {
	// Avoid retaining extremely large buffers in the pool
	if cap(p.b) <= 64<<10 {
		ioBufPool.Put(p)
	}
}

func acquireRequest() *request {
	return requestPool.Get().(*request)
}

func releaseRequest(r *request) {
	requestPool.Put(r)
}

func acquireHeader() *pdu.Header {
	return headerPool.Get().(*pdu.Header)
}

func releaseHeader(h *pdu.Header) {
	*h = pdu.Header{}
	headerPool.Put(h)
}

func acquireHeaderPacket() *pdu.HeaderPacket {
	return headerPacketPool.Get().(*pdu.HeaderPacket)
}

func releaseHeaderPacket(hp *pdu.HeaderPacket) {
	// Do not return the embedded Header here; it is managed separately.
	hp.Header = nil
	hp.Packet = nil
	headerPacketPool.Put(hp)
}
