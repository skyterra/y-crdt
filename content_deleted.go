package y_crdt

import "errors"

type ContentDeleted struct {
	Length Number
}

func (c *ContentDeleted) GetLength() Number {
	return c.Length
}

func (c *ContentDeleted) GetContent() ArrayAny {
	return nil
}

func (c *ContentDeleted) IsCountable() bool {
	return false
}

func (c *ContentDeleted) Copy() IAbstractContent {
	return NewContentDeleted(c.Length)
}

func (c *ContentDeleted) Splice(offset Number) IAbstractContent {
	if offset > c.Length {
		offset = c.Length
	}

	right := NewContentDeleted(c.Length - offset)
	c.Length = offset
	return right
}

func (c *ContentDeleted) MergeWith(right IAbstractContent) bool {
	r, ok := right.(*ContentDeleted)
	if !ok {
		return false
	}

	c.Length += r.Length
	return true
}

func (c *ContentDeleted) Integrate(trans *Transaction, item *Item) {
	AddToDeleteSet(trans.DeleteSet, item.ID.Client, item.ID.Clock, c.Length)
	item.MarkDeleted()
}

func (c *ContentDeleted) Delete(trans *Transaction) {

}

func (c *ContentDeleted) GC(store *StructStore) {

}

func (c *ContentDeleted) Write(encoder *UpdateEncoderV1, offset Number) error {
	if offset > c.Length {
		return errors.New("offset is larger than length")
	}

	encoder.WriteLen(c.Length - offset)
	return nil
}

func (c *ContentDeleted) GetRef() uint8 {
	return RefContentDeleted
}

func NewContentDeleted(length Number) *ContentDeleted {
	return &ContentDeleted{Length: length}
}

func ReadContentDeleted(decoder *UpdateDecoderV1) (IAbstractContent, error) {
	length, err := decoder.ReadLen()
	if err != nil {
		return nil, err
	}

	return NewContentDeleted(length), nil
}
