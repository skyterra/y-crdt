package y_crdt

// define the mask to get the low n bits of a byte.
const (
	BITS0 = 0
	BITS1 = 1   // low 1 bit
	BITS2 = 3   // low 2 bits
	BITS3 = 7   // low 3 bits
	BITS4 = 15  // low 4 bits
	BITS5 = 31  // low 5 bits
	BITS6 = 63  // low 6 bits
	BITS7 = 127 // low 7 bits
	BITS8 = 255 // low 8 bits
)

// define the mask to get the specific bit of a byte.
const (
	BIT1 = 1   // first bit
	BIT2 = 2   // second bit
	BIT3 = 4   // third bit
	BIT4 = 8   // fourth bit
	BIT5 = 16  // fifth bit
	BIT6 = 32  // sixth bit
	BIT7 = 64  // seventh bit
	BIT8 = 128 // eighth bit
)

const (
	KeywordUndefined = "undefined"
)

// RefContent define reference content type
const (
	RefGC             = iota // 0 GC is not ItemContent
	RefContentDeleted        // 1
	RefContentJson           // 2
	RefContentBinary         // 3
	RefContentString         // 4
	RefContentEmbed          // 5
	RefContentFormat         // 6
	RefContentType           // 7
	RefContentAny            // 8
	RefContentDoc            // 9
	RefSkip                  // 10 Skip is not ItemContent
)

// RefID define reference id
const (
	YArrayRefID = iota
	YMapRefID
	YTextRefID
	YXmlElementRefID
	YXmlFragmentRefID
	YXmlHookRefID
	YXmlTextRefID
)
