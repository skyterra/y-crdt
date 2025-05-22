package y_crdt

import "github.com/mitchellh/copystructure"

type ContentEmbed struct {
	Embed interface{}
}

func (c *ContentEmbed) GetLength() Number {
	return 1
}

func (c *ContentEmbed) GetContent() ArrayAny {
	return ArrayAny{c.Embed}
}

func (c *ContentEmbed) IsCountable() bool {
	return true
}

func (c *ContentEmbed) Copy() IAbstractContent {
	embed, err := copystructure.Copy(c.Embed)
	if err != nil {
		return nil
	}

	return NewContentEmbed(embed)
}

func (c *ContentEmbed) Splice(offset Number) IAbstractContent {
	return nil
}

func (c *ContentEmbed) MergeWith(right IAbstractContent) bool {
	return false
}

func (c *ContentEmbed) Integrate(trans *Transaction, item *Item) {

}

func (c *ContentEmbed) Delete(trans *Transaction) {

}

func (c *ContentEmbed) GC(store *StructStore) {

}

func (c *ContentEmbed) Write(encoder *UpdateEncoderV1, offset Number) error {
	return encoder.WriteJson(c.Embed)
}

func (c *ContentEmbed) GetRef() uint8 {
	return RefContentEmbed
}

func NewContentEmbed(embed interface{}) *ContentEmbed {
	return &ContentEmbed{
		Embed: embed,
	}
}

func ReadContentEmbed(decoder *UpdateDecoderV1) (IAbstractContent, error) {
	embed, err := decoder.ReadJson()
	if err != nil {
		return nil, err
	}

	return NewContentEmbed(embed), nil
}
