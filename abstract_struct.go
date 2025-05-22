package y_crdt

type IAbstractStruct interface {
	GetID() *ID
	GetLength() Number
	SetLength(length Number)
	Deleted() bool
	MergeWith(right IAbstractStruct) bool
	Write(encoder *UpdateEncoderV1, offset Number)
	Integrate(trans *Transaction, offset Number)
	GetMissing(trans *Transaction, store *StructStore) (Number, error)
}

type AbstractStruct struct {
	ID     ID
	Length Number
}

func (s *AbstractStruct) GetID() *ID {
	return &s.ID
}

func (s *AbstractStruct) GetLength() Number {
	return s.Length
}

func (s *AbstractStruct) SetLength(length Number) {
	s.Length = length
}

func (s *AbstractStruct) Deleted() bool {
	return false
}

// Merge this struct with the item to the right.
// This method is already assuming that `this.id.clock + this.length === this.id.clock`.
// Also this method does *not* remove right from StructStore!
// @param {AbstractStruct} right
// @return {boolean} whether this merged with right
func (s *AbstractStruct) MergeWith(right IAbstractStruct) bool {
	return false
}

func (s *AbstractStruct) Write(encoder *UpdateEncoderV1, offset Number) {

}

func (s *AbstractStruct) Integrate(trans *Transaction, offset Number) {

}

func (s *AbstractStruct) GetMissing(trans *Transaction, store *StructStore) (Number, error) {
	return 0, nil
}
