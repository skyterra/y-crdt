package y_crdt

import (
	"fmt"

	"github.com/mitchellh/copystructure"
)

var typeRefs = []func(decoder *UpdateDecoderV1) (IAbstractType, error){
	readYArray,
	readYMap,
	readYText,
	readYXmlElement,
	readYXmlFragment,
	readYXmlHook,
	readYXmlText,
}

type ContentType struct {
	Type IAbstractType
}

func (c *ContentType) GetLength() Number {
	return 1
}

func (c *ContentType) GetContent() ArrayAny {
	return ArrayAny{c.Type}
}

func (c *ContentType) IsCountable() bool {
	return true
}

func (c *ContentType) Copy() IAbstractContent {
	cpType, err := copystructure.Copy(c.Type)
	if err != nil {
		return nil
	}

	return NewContentType(cpType.(IAbstractType))
}

func (c *ContentType) Splice(offset Number) IAbstractContent {
	return nil
}

func (c *ContentType) MergeWith(right IAbstractContent) bool {
	return false
}

func (c *ContentType) Integrate(trans *Transaction, item *Item) {
	c.Type.Integrate(trans.Doc, item)
}

func (c *ContentType) Delete(trans *Transaction) {
	// TODO

	// item := c.Type.StartItem()
	// for item != nil {
	// 	if !item.Deleted() {
	// 		item.Delete(trans)
	// 	} else {
	// 		// Whis will be gc'd later and we want to merge it if possible
	// 		// We try to merge all deleted items after each transaction,
	// 		// but we have no knowledge about that this needs to be merged
	// 		// since it is not in transaction.ds. Hence we add it to transaction._mergeStructs
	// 		//trans._mergeStructs.push(item)
	// 	}
	// 	item = item.Right
	// }

}

func (c *ContentType) GC(store *StructStore) {
	// TODO
}

func (c *ContentType) Write(encoder *UpdateEncoderV1, offset Number) error {
	c.Type.Write(encoder)
	return nil
}

func (c *ContentType) GetRef() uint8 {
	return RefContentType
}

func NewContentType(t IAbstractType) *ContentType {
	return &ContentType{Type: t}
}

func ReadContentType(decoder *UpdateDecoderV1) (IAbstractContent, error) {
	refID, err := decoder.ReadTypeRef()
	if err != nil {
		return nil, err
	}

	if int(refID) >= len(typeRefs) {
		return nil, fmt.Errorf("index out of range. refID:%d len:%d", refID, len(typeRefs))
	}

	refType, err := typeRefs[refID](decoder)
	if err != nil {
		return nil, err
	}

	return NewContentType(refType), nil
}

func readYArray(decoder *UpdateDecoderV1) (IAbstractType, error) {
	return NewYArray(), nil
}

func readYMap(decoder *UpdateDecoderV1) (IAbstractType, error) {
	return NewYMap(nil), nil
}

func readYText(decoder *UpdateDecoderV1) (IAbstractType, error) {
	return NewYText(""), nil
}

func readYXmlElement(decoder *UpdateDecoderV1) (IAbstractType, error) {
	key, err := decoder.ReadKey()
	if err != nil {
		return nil, err
	}

	return NewYXmlElement(key), nil
}

func readYXmlFragment(decoder *UpdateDecoderV1) (IAbstractType, error) {
	return NewYXmlFragment(), nil
}

func readYXmlHook(decoder *UpdateDecoderV1) (IAbstractType, error) {
	key, err := decoder.ReadKey()
	if err != nil {
		return nil, err
	}

	return NewYXmlHook(key), nil
}

func readYXmlText(decoder *UpdateDecoderV1) (IAbstractType, error) {
	return NewYXmlText(), nil
}
