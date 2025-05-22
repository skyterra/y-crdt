package y_crdt

import "errors"

var ContentRefs = []func(*UpdateDecoderV1) (IAbstractContent, error){
	func(v1 *UpdateDecoderV1) (IAbstractContent, error) {
		return nil, errors.New("unexpected case")
	}, // GC is not ItemContent
	ReadContentDeleted, // 1
	ReadContentJson,    // 2
	ReadContentBinary,  // 3
	ReadContentString,  // 4
	ReadContentEmbed,   // 5
	ReadContentFormat,  // 6
	ReadContentType,    // 7
	ReadContentAny,     // 8
	ReadContentDoc,     // 9
	func(v1 *UpdateDecoderV1) (IAbstractContent, error) { // 10 - Skip is not ItemContent
		return nil, errors.New("unexpected case")
	},
}

type IAbstractContent interface {
	GetLength() Number
	GetContent() ArrayAny
	IsCountable() bool
	Copy() IAbstractContent
	Splice(offset Number) IAbstractContent
	MergeWith(right IAbstractContent) bool
	Integrate(trans *Transaction, item *Item)
	Delete(trans *Transaction)
	GC(store *StructStore)
	Write(encoder *UpdateEncoderV1, offset Number) error
	GetRef() uint8
}

func ReadItemContent(decoder *UpdateDecoderV1, info uint8) IAbstractContent {
	refID := int(info & BITS5)
	if refID >= len(ContentRefs) {
		Logf("[crdt] read item content failed. info:%d refID:%d err:index out of range", info, refID)
		return nil
	}

	c, err := ContentRefs[refID](decoder)
	if err != nil {
		Logf("[crdt] read item content failed. info:%d refID:%d err:%s", info, refID, err.Error())
		return nil
	}

	return c
}
