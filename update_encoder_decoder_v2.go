package y_crdt

import "bytes"

type DSEncoderV2 struct {
	RestEncoder *bytes.Buffer
	DsCurrVal   Number
}

type UpdateEncoderV2 struct {
	DSEncoderV2
}

func (v2 *DSEncoderV2) ToUint8Array() []uint8 {
	return v2.RestEncoder.Bytes()
}

func (v2 *DSEncoderV2) ResetDsCurVal() {
	v2.DsCurrVal = 0
}

func (v2 *DSEncoderV2) WriteDsClock(clock Number) {
	diff := clock - v2.DsCurrVal
	v2.DsCurrVal = clock
	WriteVarUint(v2.RestEncoder, uint64(diff))
}

func (v2 *DSEncoderV2) WriteDsLen(length Number) {
	if length == 0 {
		return
	}

	WriteVarUint(v2.RestEncoder, uint64(length-1))
	v2.DsCurrVal += length
}

func (v2 *UpdateEncoderV2) ToUint8Array() []uint8 {
	encoder := new(bytes.Buffer)
	WriteVarUint(encoder, 0)
	WriteVarUint8Array(encoder, nil)        // keyClockEncoder
	WriteVarUint8Array(encoder, nil)        // clientEncoder
	WriteVarUint8Array(encoder, nil)        // leftClockEncoder
	WriteVarUint8Array(encoder, nil)        // rightClockEncoder
	WriteVarUint8Array(encoder, nil)        // infoEncoder
	WriteUint8Array(encoder, []uint8{1, 0}) // stringEncoder
	WriteVarUint8Array(encoder, nil)        // parentInfoEncoder
	WriteVarUint8Array(encoder, nil)        // typeRefEncoder
	WriteVarUint8Array(encoder, nil)        // lenEncoder

	WriteUint8Array(encoder, v2.RestEncoder.Bytes())
	return encoder.Bytes()
}

func NewUpdateEncoderV2() *UpdateEncoderV2 {
	return &UpdateEncoderV2{
		DSEncoderV2{
			RestEncoder: new(bytes.Buffer),
			DsCurrVal:   0,
		},
	}
}
