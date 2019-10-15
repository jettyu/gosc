package bitmap

// Bitmap ...
type Bitmap struct {
	bits []byte
	max  uint
}

// NewBitmap ...
func NewBitmap(max uint) *Bitmap {
	n := int(max>>3) + 1
	return &Bitmap{
		bits: make([]byte, n),
		max:  max,
	}
}

// Set ...
func (p *Bitmap) Set(i uint) {
	index := int(i >> 3)
	pos := i & 0x07
	p.bits[index] |= 1 << pos
}

// Has ...
func (p *Bitmap) Has(i uint) bool {
	index := int(i >> 3)
	pos := i & 0x07
	return p.bits[index]&(1<<pos) != 0
}

// Del ...
func (p *Bitmap) Del(i uint) {
	index := int(i >> 3)
	pos := i & 0x07
	p.bits[index] |= 1 << pos
	p.bits[index] = p.bits[index] & ^(1 << pos)
}

// ResetMax ...
func (p *Bitmap) ResetMax(max uint) {
	p.max = max
	n := int(max>>3) + 1
	if n > len(p.bits) {
		p.bits = append(p.bits, make([]byte, n-len(p.bits))...)
		return
	}
	p.bits = p.bits[:n]
	return
}

// Max ...
func (p *Bitmap) Max() uint {
	return p.max
}
