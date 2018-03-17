package net

import (
	"optimusprime/net/ratelimiter"
	"encoding/binary"
	"io"
)

type reader struct {
	r         io.Reader
	buf       []byte
	ratelimit *ratelimiter.RateLimit
}

func newReader(r io.Reader) *reader {
	rtl := ratelimiter.NewRateLimit(defaultStrategy, defaultRate, defaultCapacity)
	return &reader{
		r:         r,
		buf:       make([]byte, defaultPacketSize),
		ratelimit: rtl,
	}
}

func (r *reader) readPacket() (packet []byte, err error) {
	//read head, get packet length
	n, err := r.readHead()
	if err != nil {
		return
	}
	// read body
	if r.ratelimit != nil {
		r.ratelimit.Stop(int64(n))
	}
	_, err = r.readBody(n)
	if err != nil {
		return
	}
	packet = make([]byte, 0, n)
	packet = append(packet, r.buf[:n]...)
	return
}

func (r *reader) readHead() (hlen int, err error) {
	n, err := r.readUint32BE()
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

func (r *reader) readUint32BE() (n uint32, err error) {
	_, err = io.ReadFull(r.r, r.buf[:defaultHeadSize])
	if err != nil {
		return 0, err
	}
	n = binary.BigEndian.Uint32(r.buf[:defaultHeadSize])
	return
}

func (r *reader) readBody(blen int) (n int, err error) {
	if blen > defaultPacketSize {
		r.buf = make([]byte, blen)
	}

	return io.ReadFull(r.r, r.buf[:blen])
}
