package y_crdt

type ContentBinary struct {
	Content []uint8
}

func (c *ContentBinary) GetLength() Number {
	return 1
}

func (c *ContentBinary) GetContent() ArrayAny {
	return ArrayAny{c.Content}
}

func (c *ContentBinary) IsCountable() bool {
	return true
}

func (c *ContentBinary) Copy() IAbstractContent {
	content := make([]uint8, 0, len(c.Content))
	for _, v := range c.Content {
		// content[i] = v
		content = append(content, v)
	}

	return NewContentBinary(content)
}

func (c *ContentBinary) Splice(offset Number) IAbstractContent {
	return nil
}

func (c *ContentBinary) MergeWith(right IAbstractContent) bool {
	return false
}

func (c *ContentBinary) Integrate(trans *Transaction, item *Item) {

}

func (c *ContentBinary) Delete(trans *Transaction) {

}

func (c *ContentBinary) GC(store *StructStore) {

}

func (c *ContentBinary) Write(encoder *UpdateEncoderV1, offset Number) error {
	encoder.WriteBuf(c.Content)
	return nil
}

func (c *ContentBinary) GetRef() uint8 {
	return RefContentBinary
}

func NewContentBinary(content []uint8) *ContentBinary {
	return &ContentBinary{
		Content: content,
	}
}

func ReadContentBinary(decoder *UpdateDecoderV1) (IAbstractContent, error) {
	content, err := decoder.ReadBuf()
	if err != nil {
		return nil, err
	}

	return NewContentBinary(content), nil
}
