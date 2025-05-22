package y_crdt

import "errors"

const StructSkipRefNumber = 10

type Skip struct {
	AbstractStruct
}

func (s *Skip) Deleted() bool {
	return true
}

func (s *Skip) Delete() {

}

func (s *Skip) MergeWith(right IAbstractStruct) bool {
	r, ok := right.(*Skip)
	if !ok {
		return false
	}

	s.Length += r.Length
	return true
}

func (s *Skip) Integrate(trans *Transaction, offset Number) {
	return
}

func (s *Skip) Write(encoder *UpdateEncoderV1, offset Number) {
	encoder.WriteInfo(StructSkipRefNumber)
	// write as VarUint because Skips can't make use of predictable length-encoding
	WriteVarUint(encoder.RestEncoder, uint64(s.Length-offset))
}

func (s *Skip) GetMissing(trans *Transaction, store *StructStore) (Number, error) {
	return 0, errors.New("gc not support this function")
}

func NewSkip(id ID, length Number) *Skip {
	return &Skip{AbstractStruct{ID: id, Length: length}}
}
