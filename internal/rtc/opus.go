package rtc

import (
	"gopkg.in/hraban/opus.v2"
)

type opusDecoder struct {
	opusd *opus.Decoder
	buffer  []byte
	samples []int16
}

func newDecoder() (*opusDecoder, error) {
	opusd, err := opus.NewDecoder(48000, 1)
	if err != nil {
		return nil, err
	}
	return &opusDecoder{
		opusd: opusd,
		buffer:  make([]byte, 2000),
		samples: make([]int16, 1000),
	}, nil
}

func (d *opusDecoder) decode(encoded []byte) ([]byte, error) {
	nsamples, err := d.opusd.Decode(encoded, d.samples)
	if err != nil {
		return nil, err
	}
	ix := 0
	for _, sample := range d.samples[:nsamples] {
		hi, lo := uint8(sample>>8), uint8(sample&0xff)
		d.buffer[ix] = lo
		d.buffer[ix+1] = hi
		ix += 2
	}
	return d.buffer[:ix], nil
}
