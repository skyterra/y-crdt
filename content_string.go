package y_crdt

import "unicode/utf16"

type ContentString struct {
	Str string
}

func (c *ContentString) GetLength() Number {
	return StringLength(c.Str)
}

func (c *ContentString) GetContent() ArrayAny {
	chars := utf16.Encode([]rune(c.Str))

	content := make(ArrayAny, 0, len(chars))
	for _, c := range chars {
		// content[i] = c
		content = append(content, c)
	}

	return content
}

func (c *ContentString) IsCountable() bool {
	return true
}

func (c *ContentString) Copy() IAbstractContent {
	return NewContentString(c.Str)
}

func (c *ContentString) Splice(offset Number) IAbstractContent {
	right := &ContentString{
		Str: StringTail(c.Str, offset),
	}

	c.Str = StringHeader(c.Str, offset)

	// Prevent encoding invalid documents because of splitting of surrogate pairs: https://github.com/yjs/yjs/issues/248
	firstCharCode, err := CharCodeAt(c.Str, offset-1)
	if err == nil && firstCharCode >= 0xD800 && firstCharCode <= 0xDBFF {
		// Last character of the left split is the start of a surrogate utf16/ucs2 pair.
		// We don't support splitting of surrogate pairs because this may lead to invalid documents.
		// Replace the invalid character with a unicode replacement character (� / U+FFFD)
		c.Str = ReplaceChar(c.Str, len(c.Str)-1, '�')

		// replace right as well
		right.Str = ReplaceChar(right.Str, 0, '�')
	}

	return right
}

func (c *ContentString) MergeWith(right IAbstractContent) bool {
	r, ok := right.(*ContentString)
	if !ok {
		return false
	}

	c.Str = MergeString(c.Str, r.Str)
	return true
}

func (c *ContentString) Integrate(trans *Transaction, item *Item) {

}

func (c *ContentString) Delete(trans *Transaction) {

}

func (c *ContentString) GC(store *StructStore) {

}

func (c *ContentString) Write(encoder *UpdateEncoderV1, offset Number) error {
	if offset == 0 {
		encoder.WriteString(c.Str)
	} else {
		encoder.WriteString(StringTail(c.Str, offset))
	}

	return nil
}

func (c *ContentString) GetRef() uint8 {
	return RefContentString
}

func NewContentString(str string) *ContentString {
	return &ContentString{
		Str: str,
	}
}

func ReadContentString(decoder *UpdateDecoderV1) (IAbstractContent, error) {
	str, err := decoder.ReadString()
	if err != nil {
		return nil, err
	}

	return NewContentString(str), nil
}
