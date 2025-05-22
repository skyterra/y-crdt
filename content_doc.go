package y_crdt

import (
	"github.com/mitchellh/copystructure"
)

const (
	OptKeyGC       = "gc"
	OptKeyAutoLoad = "autoLoad"
	OptKeyMeta     = "meta"
)

type ContentDoc struct {
	Doc  *Doc
	Opts Object
}

func (c *ContentDoc) GetLength() Number {
	return 1
}

func (c *ContentDoc) GetContent() ArrayAny {
	return ArrayAny{c.Doc}
}

func (c *ContentDoc) IsCountable() bool {
	return true
}

func (c *ContentDoc) Copy() IAbstractContent {
	doc, err := copystructure.Copy(c.Doc)
	if err != nil {
		return nil
	}

	return NewContentDoc(doc.(*Doc))
}

func (c *ContentDoc) Splice(offset Number) IAbstractContent {
	return nil
}

func (c *ContentDoc) MergeWith(right IAbstractContent) bool {
	return false
}

func (c *ContentDoc) Integrate(trans *Transaction, item *Item) {
	// TODO
}

func (c *ContentDoc) Delete(trans *Transaction) {
	// TODO
}

func (c *ContentDoc) GC(store *StructStore) {

}

func (c *ContentDoc) Write(encoder *UpdateEncoderV1, offset Number) error {
	err := encoder.WriteString(c.Doc.Guid)
	if err != nil {
		return err
	}

	encoder.WriteAny(c.Opts)
	return nil
}

func (c *ContentDoc) GetRef() uint8 {
	return RefContentDoc
}

func NewContentDoc(doc *Doc) *ContentDoc {
	c := &ContentDoc{
		Doc:  doc,
		Opts: NewObject(),
	}

	if !doc.GC {
		c.Opts[OptKeyGC] = false
	}

	if doc.AutoLoad {
		c.Opts[OptKeyAutoLoad] = true
	}

	if doc.Meta != nil {
		c.Opts[OptKeyMeta] = doc.Meta
	}

	return c
}

func ReadContentDoc(decoder *UpdateDecoderV1) (IAbstractContent, error) {
	guid, err := decoder.ReadString()
	if err != nil {
		return nil, err
	}

	any, err := decoder.ReadAny()
	if err != nil {
		return nil, err
	}

	opts := any.(Object)

	gc, ok := opts[OptKeyGC].(bool)
	if !ok {
		gc = true // 不存在gc参数，使用默认参数
	}

	autoLoad, ok := opts[OptKeyAutoLoad].(bool)
	if !ok {
		autoLoad = false // 不存在autoLoad参数，使用默认参数
	}

	doc := NewDoc(guid, gc, DefaultGCFilter, opts[OptKeyMeta], autoLoad)
	return NewContentDoc(doc), nil
}
