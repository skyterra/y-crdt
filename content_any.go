package y_crdt

import (
	"errors"

	"github.com/mitchellh/copystructure"
)

type ContentAny struct {
	Arr ArrayAny
}

func (c *ContentAny) GetLength() Number {
	return len(c.Arr)
}

func (c *ContentAny) GetContent() ArrayAny {
	return c.Arr
}

func (c *ContentAny) IsCountable() bool {
	return true
}

func (c *ContentAny) Copy() IAbstractContent {
	arr, err := copystructure.Copy(c.Arr)
	if err != nil {
		return nil
	}

	return NewContentAny(arr.(ArrayAny))
}

func (c *ContentAny) Splice(offset Number) IAbstractContent {
	right := c.Copy().(*ContentAny)
	right.Arr = right.Arr[offset:]
	c.Arr = c.Arr[:offset]
	return right
}

func (c *ContentAny) MergeWith(right IAbstractContent) bool {
	r, ok := right.(*ContentAny)
	if !ok {
		return false
	}

	c.Arr = append(c.Arr, r.Arr...)
	return true
}

func (c *ContentAny) Integrate(trans *Transaction, item *Item) {

}

func (c *ContentAny) Delete(trans *Transaction) {

}

func (c *ContentAny) GC(store *StructStore) {

}

func (c *ContentAny) Write(encoder *UpdateEncoderV1, offset Number) error {
	length := len(c.Arr)
	if offset > length {
		return errors.New("offset is larger than length")
	}

	encoder.WriteLen(length - offset)
	for i := offset; i < length; i++ {
		c := c.Arr[i]
		encoder.WriteAny(c)
	}

	return nil
}

func (c *ContentAny) GetRef() uint8 {
	return RefContentAny
}

func NewContentAny(arr ArrayAny) *ContentAny {
	return &ContentAny{Arr: arr}
}

func ReadContentAny(decoder *UpdateDecoderV1) (IAbstractContent, error) {
	length, err := decoder.ReadLen()
	if err != nil {
		return nil, err
	}

	var cs ArrayAny
	for i := 0; i < length; i++ {
		c, err := decoder.ReadAny()
		if err != nil {
			return nil, err
		}

		cs = append(cs, c)
	}

	return NewContentAny(cs), nil
}
