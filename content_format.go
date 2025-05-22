package y_crdt

import "github.com/mitchellh/copystructure"

type ContentFormat struct {
	Key   string
	Value interface{}
}

func (c *ContentFormat) GetLength() Number {
	return 1
}

func (c *ContentFormat) GetContent() ArrayAny {
	return nil
}

func (c *ContentFormat) IsCountable() bool {
	return false
}

func (c *ContentFormat) Copy() IAbstractContent {
	value, err := copystructure.Copy(c.Value)
	if err != nil {
		return nil
	}

	return NewContentFormat(c.Key, value)
}

func (c *ContentFormat) Splice(offset Number) IAbstractContent {
	// 不支持
	return nil
}

func (c *ContentFormat) MergeWith(right IAbstractContent) bool {
	return false
}

func (c *ContentFormat) Integrate(trans *Transaction, item *Item) {
	// todo searchmarker are currently unsupported for rich text documents
	(item.Parent).(IAbstractType).SetSearchMarker(nil)
}

func (c *ContentFormat) Delete(trans *Transaction) {

}

func (c *ContentFormat) GC(store *StructStore) {

}

func (c *ContentFormat) Write(encoder *UpdateEncoderV1, offset Number) error {
	encoder.WriteKey(c.Key)
	encoder.WriteJson(c.Value)
	return nil
}

func (c *ContentFormat) GetRef() uint8 {
	return RefContentFormat
}

func NewContentFormat(key string, value interface{}) *ContentFormat {
	return &ContentFormat{
		Key:   key,
		Value: value,
	}
}

func ReadContentFormat(decoder *UpdateDecoderV1) (IAbstractContent, error) {
	key, err := decoder.ReadString()
	if err != nil {
		return nil, err
	}

	value, err := decoder.ReadJson()
	if err != nil {
		return nil, err
	}

	return NewContentFormat(key, value), nil
}
