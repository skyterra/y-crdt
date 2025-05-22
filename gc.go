package y_crdt

import "errors"

const (
	StructGCRefNumber = 0
)

type GC struct {
	AbstractStruct
}

func (gc *GC) Deleted() bool {
	return true
}

func (gc *GC) Delete() {

}

func (gc *GC) MergeWith(right IAbstractStruct) bool {
	r, ok := right.(*GC)
	if !ok {
		return false
	}

	gc.Length += r.Length
	return true
}

func (gc *GC) Integrate(trans *Transaction, offset Number) {
	if offset > 0 {
		gc.ID.Clock += offset
		gc.Length -= offset
	}

	err := AddStruct(trans.Doc.Store, gc)
	if err != nil {
		Logf("[crdt] %s.", err.Error())
	}
}

func (gc *GC) Write(encoder *UpdateEncoderV1, offset Number) {
	encoder.WriteInfo(StructGCRefNumber)
	encoder.WriteLen(gc.Length - offset)
}

func (gc *GC) GetMissing(trans *Transaction, store *StructStore) (Number, error) {
	return 0, errors.New("gc not support this function")
}

func NewGC(id ID, length Number) *GC {
	return &GC{
		AbstractStruct{ID: id, Length: length},
	}
}
