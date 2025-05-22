package y_crdt

import (
	"bytes"
	"errors"
	"sort"
)

const (
	RecordPositionUnit = 1024
)

var (
	ErrInvalidData = errors.New("invalid data error")
)

type CurrWrite struct {
	S      IAbstractStruct
	Offset Number
}

type LazyStructReader struct {
	Gen         func() IAbstractStruct
	Curr        IAbstractStruct
	Done        bool
	FilterSkips bool
}

func (r *LazyStructReader) Next() IAbstractStruct {
	// ignore "Skip" structs
	r.Curr = r.Gen()
	for r.FilterSkips && r.Curr != nil && IsSameType(r.Curr, &Skip{}) {
		r.Curr = r.Gen()
	}

	return r.Curr
}

type PositionInfo struct {
	Clock     Number
	StartByte int
	StructNo  int
}

type ClientStruct struct {
	Written      Number
	RestEncoder  []uint8
	PositionList []PositionInfo
	Client       Number
	Start        int
	End          int
}

type LazyStructWriter struct {
	CurrClient Number
	StartClock Number
	Written    Number
	Encoder    *UpdateEncoderV1

	// We want to write operations lazily, but also we need to know beforehand how many operations we want to write for each client.
	//
	//  This kind of meta-information (#clients, #structs-per-client-written) is written to the restEncoder.
	//
	//  We fragment the restEncoder and store a slice of it per-client until we know how many clients there are.
	//  When we flush (toUint8Array) we write the restEncoder using the fragments and the meta-information.
	ClientStructs      []ClientStruct
	NeedRecordPosition bool
	PositionList       []PositionInfo
}

func NewLazyStructReader(decoder *UpdateDecoderV1, filterSkips bool, stopIfError bool) *LazyStructReader {
	r := &LazyStructReader{
		Gen:         CreateLazyStructReaderGenerator(decoder, stopIfError).Next(),
		FilterSkips: filterSkips,
		Done:        false,
	}

	r.Next()
	return r
}

func NewLazyStructWriter(encoder *UpdateEncoderV1) *LazyStructWriter {
	return &LazyStructWriter{
		Encoder: encoder,
	}
}

func LogUpdate(update []uint8, YDecoder func([]byte) *UpdateDecoderV1) {
	LogUpdateV2(update, YDecoder)
}

func LogUpdateV2(update []uint8, YDecoder func([]byte) *UpdateDecoderV1) {
	var structs []IAbstractStruct
	updateDecoder := YDecoder(update)

	lazyDecoder := NewLazyStructReader(updateDecoder, false, false)
	for curr := lazyDecoder.Curr; curr != nil; curr = lazyDecoder.Next() {
		structs = append(structs, curr)
	}

	Logf("[crdt] Structs: %v", structs)
	ds := ReadDeleteSet(updateDecoder)
	Logf("[crdt] DeleteSet: %+v", ds)
}

func MergeUpdates(updates [][]uint8, YDecoder func([]byte) *UpdateDecoderV1, YEncoder func() *UpdateEncoderV1, stopIfError bool) []uint8 {
	return MergeUpdatesV2(updates, YDecoder, YEncoder, stopIfError)
}

func EncodeStateVectorFromUpdateV2(update []uint8, YEncoder func() *UpdateEncoderV1, YDecoder func([]byte) *UpdateDecoderV1) []uint8 {
	encoder := YEncoder()
	updateDecoder := NewLazyStructReader(YDecoder(update), false, false)
	curr := updateDecoder.Curr
	if curr != nil {
		size := 0
		currClient := curr.GetID().Client
		stopCounting := curr.GetID().Clock != 0 // must start at 0
		var currClock Number
		if !stopCounting {
			currClock = curr.GetID().Clock + curr.GetLength()
		}

		for ; curr != nil; curr = updateDecoder.Next() {
			if currClient != curr.GetID().Client {
				if currClock != 0 {
					size++
					// We found a new client
					// write what we have to the encoder
					WriteVarUint(encoder.RestEncoder, uint64(currClient))
					WriteVarUint(encoder.RestEncoder, uint64(currClock))

				}

				currClient = curr.GetID().Client
				currClock = 0
				stopCounting = curr.GetID().Clock != 0
			}

			// we ignore skips
			if IsSameType(curr, &Skip{}) {
				stopCounting = true
			}

			if !stopCounting {
				currClock = curr.GetID().Clock + curr.GetLength()
			}
		}

		// write what we have
		if currClock != 0 {
			size++
			WriteVarUint(encoder.RestEncoder, uint64(currClient))
			WriteVarUint(encoder.RestEncoder, uint64(currClock))
		}

		// prepend the size of the state vector
		enc := NewUpdateEncoderV1()
		WriteVarUint(enc.RestEncoder, uint64(size))
		WriteUint8Array(enc.RestEncoder, encoder.RestEncoder.Bytes())
		return enc.ToUint8Array()
	} else {
		WriteVarUint(encoder.RestEncoder, 0)
		return encoder.ToUint8Array()
	}
}

func EncodeStateVectorFromUpdate(update []uint8) []uint8 {
	return EncodeStateVectorFromUpdateV2(update, NewUpdateEncoderV1, NewUpdateDecoderV1)
}

func ParseUpdateMetaV2(update []uint8, YDecoder func([]byte) *UpdateDecoderV1) (map[Number]Number, map[Number]Number) {
	from := make(map[Number]Number)
	to := make(map[Number]Number)

	updateDecoder := NewLazyStructReader(YDecoder(update), false, false)
	curr := updateDecoder.Curr

	if curr != nil {
		currClient := curr.GetID().Client
		currClock := curr.GetID().Clock
		// write the beginning to `from`
		from[currClient] = currClock
		for ; curr != nil; curr = updateDecoder.Next() {
			if currClient != curr.GetID().Client {
				// We found a new client
				// write the end to `to`
				to[currClient] = currClock
				// write the beginning to `from`
				from[curr.GetID().Client] = curr.GetID().Clock
				currClient = curr.GetID().Client
			}
			currClock = curr.GetID().Clock + curr.GetLength()
		}
		// write the end to `to`
		to[currClient] = currClock
	}
	return from, to
}

func ParseUpdateMeta(update []uint8) (map[Number]Number, map[Number]Number) {
	return ParseUpdateMetaV2(update, NewUpdateDecoderV1)
}

func SliceStruct(left IAbstractStruct, diff Number) IAbstractStruct {
	client, clock := left.GetID().Client, left.GetID().Clock

	if IsSameType(left, &GC{}) {
		return NewGC(GenID(client, clock+diff), left.GetLength()-diff)
	}

	if IsSameType(left, &Skip{}) {
		return NewSkip(GenID(client, clock+diff), left.GetLength()-diff)
	}

	leftItem := left.(*Item)
	originID := GenID(client, clock+diff-1)
	parent, _ := leftItem.Parent.(IAbstractType)
	return NewItem(
		GenID(client, clock+diff),
		nil,
		&originID,
		nil,
		leftItem.RightOrigin,
		parent,
		leftItem.ParentSub,
		leftItem.Content.Splice(diff),
	)
}

// InsertionSort 只将第一个元素重新插入合适的位置，即，除第一个元素外，其他元素是有序的
func InsertionSort(a []*LazyStructReader) {
	reader := a[0]
	i := 1

	less := func(r1 *LazyStructReader, r2 *LazyStructReader) bool {
		client1 := r1.Curr.GetID().Client
		client2 := r2.Curr.GetID().Client

		if client1 == client2 {
			clockDiff := r1.Curr.GetID().Clock - r2.Curr.GetID().Clock
			if clockDiff == 0 {
				// @todo remove references to skip since the structDecoders must filter Skips.
				if IsSameType(r1, r2) {
					return false
				} else {
					return IsSameType(r1.Curr, &Skip{}) // we are filtering skips anyway.
				}
			} else {
				return clockDiff < 0
			}
		} else {
			return client1 > client2
		}
	}

	for ; i < len(a) && less(a[i], reader); i++ {
		a[i-1] = a[i]
	}

	a[i-1] = reader
}

func MergeUpdatesV2(updates [][]uint8, YDecoder func([]byte) *UpdateDecoderV1, YEncoder func() *UpdateEncoderV1, stopIfError bool) []uint8 {
	// 不要求严格检测错误时，一条update不需要走合并流程
	if len(updates) == 1 && !stopIfError {
		return updates[0]
	}

	updateDecoders := make([]*UpdateDecoderV1, 0, len(updates))
	lazyStructDecoders := make([]*LazyStructReader, 0, len(updateDecoders))

	for _, update := range updates {
		decoder := YDecoder(update)
		updateDecoders = append(updateDecoders, decoder)
		lazyStructDecoders = append(lazyStructDecoders, NewLazyStructReader(decoder, true, stopIfError))
	}

	// todo we don't need offset because we always slice before
	var currWrite *CurrWrite
	updateEncoder := NewUpdateEncoderV1()
	// write structs lazily
	lazyStructEncoder := NewLazyStructWriter(updateEncoder)

	// Note: We need to ensure that all lazyStructDecoders are fully consumed
	// Note: Should merge document updates whenever possible - even from different updates
	// Note: Should handle that some operations cannot be applied yet ()

	// Write higher clients first ⇒ sort by clientID & clock and remove decoders without content
	for i := len(lazyStructDecoders) - 1; i >= 0; i-- {
		if lazyStructDecoders[i].Curr == nil {
			lazyStructDecoders = append(lazyStructDecoders[:i], lazyStructDecoders[i+1:]...)
		}
	}

	sort.Slice(lazyStructDecoders, func(i, j int) bool {
		dec1 := lazyStructDecoders[i]
		dec2 := lazyStructDecoders[j]

		client1 := dec1.Curr.GetID().Client
		client2 := dec2.Curr.GetID().Client

		if client1 == client2 {
			clockDiff := dec1.Curr.GetID().Clock - dec2.Curr.GetID().Clock
			if clockDiff == 0 {
				// @todo remove references to skip since the structDecoders must filter Skips.
				if IsSameType(dec1, dec2) {
					return false
				} else {
					return IsSameType(dec1.Curr, &Skip{}) // we are filtering skips anyway.
				}
			} else {
				return clockDiff < 0
			}
		} else {
			return client1 > client2
		}
	})

	for true {
		// Write higher clients first ⇒ sort by clientID & clock and remove decoders without content
		for i := len(lazyStructDecoders) - 1; i >= 0; i-- {
			if lazyStructDecoders[i].Curr == nil {
				lazyStructDecoders = append(lazyStructDecoders[:i], lazyStructDecoders[i+1:]...)
			}
		}

		if len(lazyStructDecoders) == 0 {
			break
		}

		// the elements are sorted except for the first element, so we only need to insert the first element into the correct position
		InsertionSort(lazyStructDecoders)

		currDecoder := lazyStructDecoders[0]
		// write from currDecoder until the next operation is from another client or if filler-struct
		// then we need to reorder the decoders and find the next operation to write
		firstClient := currDecoder.Curr.GetID().Client
		if currWrite != nil {
			curr := currDecoder.Curr
			iterated := false

			// iterate until we find something that we haven't written already
			// remember: first the high client-ids are written
			for curr != nil &&
				curr.GetID().Clock+curr.GetLength() <= currWrite.S.GetID().Clock+currWrite.S.GetLength() &&
				curr.GetID().Client >= currWrite.S.GetID().Client {

				curr = currDecoder.Next()
				iterated = true
			}

			if curr == nil || // current decoder is empty
				curr.GetID().Client != firstClient || // check whether there is another decoder that has has updates from `firstClient`
				(iterated && curr.GetID().Clock > currWrite.S.GetID().Clock+currWrite.S.GetLength()) { // the above while loop was used and we are potentially missing updates
				continue
			}

			if firstClient != currWrite.S.GetID().Client {
				WriteStructToLazyStructWriter(lazyStructEncoder, currWrite.S, currWrite.Offset)
				currWrite = &CurrWrite{
					S:      curr,
					Offset: 0,
				}

				currDecoder.Next()
			} else {
				if currWrite.S.GetID().Clock+currWrite.S.GetLength() < curr.GetID().Clock {
					// todo write currStruct & set currStruct = Skip(clock = currStruct.id.clock + currStruct.length, length = curr.id.clock - self.clock)
					if IsSameType(currWrite.S, &Skip{}) {
						// extend existing skip
						currWrite.S.SetLength(curr.GetID().Clock + curr.GetLength() - currWrite.S.GetID().Clock)
					} else {
						WriteStructToLazyStructWriter(lazyStructEncoder, currWrite.S, currWrite.Offset)
						diff := curr.GetID().Clock - currWrite.S.GetID().Clock - currWrite.S.GetLength()
						s := NewSkip(GenID(firstClient, currWrite.S.GetID().Clock+currWrite.S.GetLength()), diff)
						currWrite = &CurrWrite{
							S:      s,
							Offset: 0,
						}
					}
				} else { // if (currWrite.struct.id.clock + currWrite.struct.length >= curr.id.clock) {
					diff := currWrite.S.GetID().Clock + currWrite.S.GetLength() - curr.GetID().Clock
					if diff > 0 {
						if IsSameType(currWrite.S, &Skip{}) {
							// prefer to slice Skip because the other struct might contain more information
							currWrite.S.SetLength(currWrite.S.GetLength() - diff)
						} else {
							curr = SliceStruct(curr, diff)
						}
					}

					if !currWrite.S.MergeWith(curr) {
						WriteStructToLazyStructWriter(lazyStructEncoder, currWrite.S, currWrite.Offset)
						currWrite = &CurrWrite{
							S:      curr,
							Offset: 0,
						}
						currDecoder.Next()
					}
				}
			}
		} else {
			currWrite = &CurrWrite{
				S:      currDecoder.Curr,
				Offset: 0,
			}
			currDecoder.Next()
		}

		for next := currDecoder.Curr; next != nil && next.GetID().Client == firstClient &&
			next.GetID().Clock == currWrite.S.GetID().Clock+currWrite.S.GetLength() &&
			!IsSameType(next, &Skip{}); next = currDecoder.Next() {
			WriteStructToLazyStructWriter(lazyStructEncoder, currWrite.S, currWrite.Offset)
			currWrite = &CurrWrite{
				S:      next,
				Offset: 0,
			}
		}
	}
	if currWrite != nil {
		WriteStructToLazyStructWriter(lazyStructEncoder, currWrite.S, currWrite.Offset)
		currWrite = nil
	}

	FinishLazyStructWriting(lazyStructEncoder)

	dss := make([]*DeleteSet, 0, len(updateDecoders))
	for _, decoder := range updateDecoders {
		dss = append(dss, ReadDeleteSet(decoder))
	}

	ds := MergeDeleteSets(dss)
	WriteDeleteSet(updateEncoder, ds)
	return updateEncoder.ToUint8Array()
}

func GenerateUpdates(lazyWriter *LazyStructWriter, maxUpdateSize int) [][]uint8 {
	updates := make([][]uint8, 0)
	for {
		update := GenerateUpdate(lazyWriter, maxUpdateSize)
		if nil == update {
			break
		}
		updates = append(updates, update)
	}
	return updates
}

func GenerateUpdate(lazyWriter *LazyStructWriter, maxUpdateSize int) []uint8 {
	clientCnt := 0
	for i := 0; i < len(lazyWriter.ClientStructs); i++ {
		partStructs := &lazyWriter.ClientStructs[i]
		if partStructs.End < len(partStructs.PositionList) {
			clientCnt++
		}
		partStructs.Start = partStructs.End
	}

	if clientCnt <= 0 {
		return nil
	}

	totalSize := 0
	for {
		totalSize = 0
		isFinished := true
		for i := 0; i < len(lazyWriter.ClientStructs); i++ {
			partStructs := &lazyWriter.ClientStructs[i]
			if partStructs.End < len(partStructs.PositionList)-1 {
				partStructs.End++
				totalSize += partStructs.PositionList[partStructs.End].StartByte - partStructs.PositionList[partStructs.Start].StartByte
				isFinished = false
			} else if partStructs.Start < len(partStructs.PositionList) {
				totalSize += len(partStructs.RestEncoder) - partStructs.PositionList[partStructs.Start].StartByte
				if partStructs.End < len(partStructs.PositionList) {
					partStructs.End++
					isFinished = false
				}
			}
		}
		if isFinished || totalSize >= maxUpdateSize {
			break
		}
	}

	// data format：update_count count/client/clock ... count/client/clock ... count/client/clock ds
	updateEncoder := NewUpdateEncoderV1()
	updateEncoder.RestEncoder.Grow(totalSize + 8*(1+clientCnt*3) + 1)
	WriteVarUint(updateEncoder.RestEncoder, uint64(clientCnt))
	for i := 0; i < len(lazyWriter.ClientStructs); i++ {
		partStructs := lazyWriter.ClientStructs[i]
		if partStructs.End <= partStructs.Start {
			continue
		}
		structCnt := 0
		var data []uint8
		if partStructs.End >= len(partStructs.PositionList) {
			structCnt = partStructs.Written - partStructs.PositionList[partStructs.Start].StructNo
			data = partStructs.RestEncoder[partStructs.PositionList[partStructs.Start].StartByte:]
		} else {
			structCnt = partStructs.PositionList[partStructs.End].StructNo - partStructs.PositionList[partStructs.Start].StructNo
			data = partStructs.RestEncoder[partStructs.PositionList[partStructs.Start].StartByte:partStructs.PositionList[partStructs.End].StartByte]
		}

		WriteVarUint(updateEncoder.RestEncoder, uint64(structCnt))
		updateEncoder.WriteClient(partStructs.Client)
		WriteVarUint(updateEncoder.RestEncoder, uint64(partStructs.PositionList[partStructs.Start].Clock))
		WriteUint8Array(updateEncoder.RestEncoder, data)
	}

	return updateEncoder.RestEncoder.Bytes()
}

func DiffUpdatesV2(update []uint8, sv []uint8, YDecoder func([]byte) *UpdateDecoderV1, YEncoder func() *UpdateEncoderV1, maxUpdateSize int) [][]uint8 {
	updates := make([][]uint8, 0)

	if len(update) <= maxUpdateSize {
		updates = append(updates, update)
		return updates
	}

	state := DecodeStateVector(sv)
	encoder := YEncoder()
	lazyStructWriter := NewLazyStructWriter(encoder)
	lazyStructWriter.NeedRecordPosition = true
	decoder := YDecoder(update)
	reader := NewLazyStructReader(decoder, false, false)
	for reader.Curr != nil {
		curr := reader.Curr
		currClient := curr.GetID().Client
		svClock := state[currClient]
		if IsSameType(reader.Curr, &Skip{}) {
			// the first written struct shouldn't be a skip
			reader.Next()
			continue
		}

		if curr.GetID().Clock+curr.GetLength() > svClock {
			WriteStructToLazyStructWriter(lazyStructWriter, curr, Max(svClock-curr.GetID().Clock, 0))
			reader.Next()

			for reader.Curr != nil && reader.Curr.GetID().Client == currClient {
				WriteStructToLazyStructWriter(lazyStructWriter, reader.Curr, 0)
				reader.Next()
			}
		} else {
			// read until something new comes up
			for reader.Curr != nil && reader.Curr.GetID().Client == currClient && reader.Curr.GetID().Clock+reader.Curr.GetLength() <= svClock {
				reader.Next()
			}
		}
	}

	FlushLazyStructWriter(lazyStructWriter)

	updates = GenerateUpdates(lazyStructWriter, maxUpdateSize)

	if len(updates) > 0 {
		// ds only stores the clock and length of items, and will be merged after transaction, so the length will not be too long, temporarily not optimized
		dsBytes := decoder.RestDecoder.Bytes()
		updates[0] = append(updates[0], dsBytes...)
		for i := 1; i < len(updates); i++ {
			updates[i] = append(updates[i], 0)
		}
	}

	return updates
}

func DiffUpdateV2(update []uint8, sv []uint8, YDecoder func([]byte) *UpdateDecoderV1, YEncoder func() *UpdateEncoderV1) []uint8 {
	state := DecodeStateVector(sv)
	encoder := YEncoder()
	lazyStructWriter := NewLazyStructWriter(encoder)
	decoder := YDecoder(update)
	reader := NewLazyStructReader(decoder, false, false)

	for reader.Curr != nil {
		curr := reader.Curr
		currClient := curr.GetID().Client
		svClock := state[currClient]
		if IsSameType(reader.Curr, &Skip{}) {
			// the first written struct shouldn't be a skip
			reader.Next()
			continue
		}

		if curr.GetID().Clock+curr.GetLength() > svClock {
			WriteStructToLazyStructWriter(lazyStructWriter, curr, Max(svClock-curr.GetID().Clock, 0))
			reader.Next()

			for reader.Curr != nil && reader.Curr.GetID().Client == currClient {
				WriteStructToLazyStructWriter(lazyStructWriter, reader.Curr, 0)
				reader.Next()
			}
		} else {
			// read until something new comes up
			for reader.Curr != nil && reader.Curr.GetID().Client == currClient && reader.Curr.GetID().Clock+reader.Curr.GetLength() <= svClock {
				reader.Next()
			}
		}
	}
	FinishLazyStructWriting(lazyStructWriter)

	// write ds
	ds := ReadDeleteSet(decoder)
	WriteDeleteSet(encoder, ds)
	return encoder.ToUint8Array()
}

func DiffUpdate(update []uint8, sv []uint8) []uint8 {
	return DiffUpdateV2(update, sv, NewUpdateDecoderV1, NewUpdateEncoderV1)
}

func DiffUpdates(update []uint8, sv []uint8, maxUpdateSize int) [][]uint8 {
	return DiffUpdatesV2(update, sv, NewUpdateDecoderV1, NewUpdateEncoderV1, maxUpdateSize)
}

func FlushLazyStructWriter(lazyWriter *LazyStructWriter) {
	if lazyWriter.Written > 0 {
		lazyWriter.ClientStructs = append(lazyWriter.ClientStructs, ClientStruct{
			Written:     lazyWriter.Written,
			RestEncoder: lazyWriter.Encoder.ToUint8Array(),
		})

		if lazyWriter.NeedRecordPosition {
			lazyWriter.ClientStructs[len(lazyWriter.ClientStructs)-1].PositionList = lazyWriter.PositionList
			lazyWriter.ClientStructs[len(lazyWriter.ClientStructs)-1].Client = lazyWriter.CurrClient
			lazyWriter.PositionList = make([]PositionInfo, 0)
		}
		lazyWriter.Encoder.RestEncoder = new(bytes.Buffer)
		lazyWriter.Written = 0
	}
}

func WriteStructToLazyStructWriter(lazyWriter *LazyStructWriter, s IAbstractStruct, offset Number) {
	// flush curr if we start another client
	if lazyWriter.Written > 0 && lazyWriter.CurrClient != s.GetID().Client {
		FlushLazyStructWriter(lazyWriter)
	}

	if lazyWriter.Written == 0 {
		lazyWriter.CurrClient = s.GetID().Client
		// write next client
		lazyWriter.Encoder.WriteClient(s.GetID().Client)

		// write startClock
		WriteVarUint(lazyWriter.Encoder.RestEncoder, uint64(s.GetID().Clock+offset))

		// record position of first struct
		if lazyWriter.NeedRecordPosition {
			pos := PositionInfo{Clock: s.GetID().Clock + offset, StartByte: lazyWriter.Encoder.RestEncoder.Len(), StructNo: lazyWriter.Written}
			lazyWriter.PositionList = append(lazyWriter.PositionList, pos)
		}
	}

	var startByte int
	if lazyWriter.NeedRecordPosition && lazyWriter.Written > 0 {
		startByte = lazyWriter.Encoder.RestEncoder.Len()
	}

	s.Write(lazyWriter.Encoder, offset)

	if lazyWriter.NeedRecordPosition && lazyWriter.Written > 0 {
		lastStartByte := lazyWriter.PositionList[len(lazyWriter.PositionList)-1].StartByte
		if lazyWriter.Encoder.RestEncoder.Len() >= lastStartByte+RecordPositionUnit {
			pos := PositionInfo{Clock: s.GetID().Clock + offset, StartByte: startByte, StructNo: lazyWriter.Written}
			lazyWriter.PositionList = append(lazyWriter.PositionList, pos)
		}
	}

	lazyWriter.Written++
}

// Call this function when we collected all parts and want to
// put all the parts together. After calling this method,
// you can continue using the UpdateEncoder.
func FinishLazyStructWriting(lazyWriter *LazyStructWriter) {
	FlushLazyStructWriter(lazyWriter)

	// this is a fresh encoder because we called flushCurr
	restEncoder := lazyWriter.Encoder.RestEncoder

	// Now we put all the fragments together.
	// This works similarly to `writeClientsStructs`

	// write # states that were updated - i.e. the clients
	WriteVarUint(restEncoder, uint64(len(lazyWriter.ClientStructs)))

	for i := 0; i < len(lazyWriter.ClientStructs); i++ {
		partStructs := lazyWriter.ClientStructs[i]

		// Works similarly to `writeStructs`

		// write # encoded structs
		WriteVarUint(restEncoder, uint64(partStructs.Written))

		// write the rest of the fragment
		WriteUint8Array(restEncoder, partStructs.RestEncoder)
	}
}

type LazyStructReaderGenerator struct {
	decoder     *UpdateDecoderV1
	stopIfError bool
}

func (l LazyStructReaderGenerator) Next() func() IAbstractStruct {
	numOfStateUpdates, numberOfStructs := -1, -1
	i, j := 0, 0
	client, clock := 0, 0
	return func() IAbstractStruct {
		if numOfStateUpdates < 0 {
			value, _ := readVarUint(l.decoder.RestDecoder)
			numOfStateUpdates = Number(value.(uint64))
		}

		var s IAbstractStruct
		innerBreak := false // mark whether the loop is terminated by break or the end of the loop
		for ; i < numOfStateUpdates; i++ {
			if numberOfStructs < 0 {
				v, _ := readVarUint(l.decoder.RestDecoder)
				numberOfStructs = Number(v.(uint64))
				client, _ = l.decoder.ReadClient()

				v, _ = readVarUint(l.decoder.RestDecoder)
				clock = Number(v.(uint64))
			}

			for ; j < numberOfStructs; j++ {
				info, _ := l.decoder.ReadInfo()
				if info == StructSkipRefNumber {
					v, _ := readVarUint(l.decoder.RestDecoder)
					length := Number(v.(uint64))
					s = NewSkip(GenID(client, clock), length)
					clock += length
					innerBreak = true
					break
				} else if BITS5&info != StructGCRefNumber {
					// If parent = null and neither left nor right are defined, then we know that `parent` is child of `y`
					// and we read the next string as parentYKey.
					// It indicates how we store/retrieve parent from `y.share`

					cantCopyParentInfo := info&(BIT7|BIT8) == 0
					var origin *ID
					if info&BIT8 == BIT8 {
						origin, _ = l.decoder.ReadLeftID()
					}

					var rightOrigin *ID
					if info&BIT7 == BIT7 {
						rightOrigin, _ = l.decoder.ReadRightID()
					}

					var parent IAbstractType
					if cantCopyParentInfo {
						ok, _ := l.decoder.ReadParentInfo()
						if ok {
							str, _ := l.decoder.ReadString()
							parent = NewYString(str)
						} else {
							parent, _ = l.decoder.ReadLeftID()
						}
					}

					var parentSub string
					if cantCopyParentInfo && info&BIT6 == BIT6 {
						parentSub, _ = l.decoder.ReadString()
					}

					s = NewItem(GenID(client, clock), nil, origin, nil, rightOrigin, parent, parentSub, ReadItemContent(l.decoder, info))
					item, _ := s.(*Item)
					if nil != item {
						clock += item.Length
						innerBreak = true
						break
					}

					if l.stopIfError {
						panic(ErrInvalidData)
					}
				} else {
					length, _ := l.decoder.ReadLen()
					if 0 == length {
						continue
					}
					s = NewGC(GenID(client, clock), length)
					clock += length
					innerBreak = true
					break
				}
			}

			if innerBreak {
				j++
				if j == numberOfStructs {
					i++
					j = 0
					numberOfStructs = -1
				}

				return s
			}

			j = 0
			numberOfStructs = -1
		}

		return nil
	}
}

func CreateLazyStructReaderGenerator(decoder *UpdateDecoderV1, stopIfError bool) LazyStructReaderGenerator {
	generator := LazyStructReaderGenerator{decoder: decoder, stopIfError: stopIfError}
	return generator
}
