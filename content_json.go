package y_crdt

import (
	"encoding/json"

	"github.com/mitchellh/copystructure"
)

type ContentJson struct {
	Arr ArrayAny
}

func (c *ContentJson) GetLength() Number {
	return len(c.Arr)
}

func (c *ContentJson) GetContent() ArrayAny {
	return c.Arr
}

func (c *ContentJson) IsCountable() bool {
	return true
}

func (c *ContentJson) Copy() IAbstractContent {
	data, err := copystructure.Copy(c.Arr)
	if err != nil {
		return nil
	}

	return NewContentJson(data.(ArrayAny))
}

func (c *ContentJson) Splice(offset Number) IAbstractContent {
	cp := c.Copy()
	if cp == nil {
		return nil
	}

	c.Arr = c.Arr[:offset]

	right := cp.(*ContentJson)
	right.Arr = right.Arr[offset:]

	return right
}

func (c *ContentJson) MergeWith(right IAbstractContent) bool {
	r, ok := right.(*ContentJson)
	if !ok {
		return false
	}

	c.Arr = append(c.Arr, r.Arr...)
	return true
}

func (c *ContentJson) Integrate(trans *Transaction, item *Item) {

}

func (c *ContentJson) Delete(trans *Transaction) {

}

func (c *ContentJson) GC(store *StructStore) {

}

func (c *ContentJson) Write(encoder *UpdateEncoderV1, offset Number) error {
	length := len(c.Arr)
	encoder.WriteLen(length - offset)
	for i := offset; i < length; i++ {
		e := c.Arr[i]

		if IsUndefined(e) {
			encoder.WriteString(KeywordUndefined)
			continue
		}

		data, err := json.Marshal(e)
		if err != nil {
			return err
		}

		encoder.WriteString(string(data))
	}

	return nil
}

func (c *ContentJson) GetRef() uint8 {
	return RefContentJson
}

func NewContentJson(arr ArrayAny) *ContentJson {
	return &ContentJson{
		Arr: arr,
	}
}

func ReadContentJson(decoder *UpdateDecoderV1) (IAbstractContent, error) {
	length, err := decoder.ReadLen()
	if err != nil {
		return nil, err
	}

	var cs ArrayAny
	for i := 0; i < length; i++ {
		c, err := decoder.ReadString()
		if err != nil {
			return nil, err
		}

		if c == KeywordUndefined {
			cs = append(cs, Undefined)
		} else {
			var obj interface{}
			err = json.Unmarshal([]byte(c), &obj)
			if err != nil {
				return nil, err
			}

			cs = append(cs, obj)
		}
	}

	return NewContentJson(cs), nil
}
