package y_crdt

import (
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode/utf16"
)

var DefaultGCFilter = func(item *Item) bool {
	return true
}

var Logf = func(format string, a ...interface{}) {
	fmt.Printf(format+"\n", a...)
}

var Log = func(a ...interface{}) {
	fmt.Println(a...)
}

// SpliceStruct inserts elements into a slice at the specified start index,
// deleting deleteCount elements if deleteCount is greater than 0.
// It returns the deleted elements if deleteCount is greater than 0, otherwise it returns nil.
func SpliceStruct(ss *[]IAbstractStruct, start Number, deleteCount Number, elements []IAbstractStruct) {
	// check if the capacity is enough to store the elements, if the deleted elements are greater than or equal to the elements to be inserted,
	// then the elements to be deleted can be directly overwritten, and the remaining elements can be moved forward to the position of the elements to be deleted.
	if deleteCount >= len(elements) {
		// find the position to insert the elements, and directly overwrite the elements to be deleted
		partSlice := (*ss)[:start]
		partSlice = append(partSlice, elements...)

		// move the remaining elements forward to position of the elements to be deleted.
		first := start + len(elements)
		second := start + deleteCount
		for i := second; i < len(*ss); i++ {
			(*ss)[first] = (*ss)[second]
			first++
			second++
		}

		*ss = (*ss)[:first]
		return
	}

	// if the capacity is not enough to store the elements, then the elements need to be copied
	// and the capacity is expanded to the sum of the original capacity and the length of the elements to be inserted.
	if cap(*ss) < (len(*ss) + len(elements) - deleteCount) {
		SpliceStructInner(ss, start, deleteCount, elements)
		return
	}

	// the capacity is enough to store the elements, then the elements need to be moved forward to the position of the elements to be deleted,
	// and then the elements to be inserted are appended to the slice.
	originLength := len(*ss)
	(*ss) = (*ss)[:len(*ss)+len(elements)-deleteCount]

	offset := len(elements) - deleteCount
	for i := originLength - 1; i >= start; i-- {
		(*ss)[i+offset] = (*ss)[i]
	}

	partSlice := (*ss)[:start]
	partSlice = append(partSlice, elements...)

	return
}

// SpliceStructInner copies the elements to be inserted into a new slice,
// and then appends the new slice to the original slice.
// It returns the deleted elements if deleteCount is greater than 0, otherwise it returns nil.
// The capacity of the original slice is not expanded.
func SpliceStructInner(ss *[]IAbstractStruct, start Number, deleteCount Number, elements []IAbstractStruct) {
	combine := make([]IAbstractStruct, 0, len(*ss)+len(elements)-deleteCount)
	combine = append(combine, (*ss)[:start]...)
	combine = append(combine, elements...)
	combine = append(combine, (*ss)[start+deleteCount:]...)
	*ss = combine

	return
}

// SpliceArray inserts elements into a slice at the specified start index,
// deleting deleteCount elements if deleteCount is greater than 0.
// It returns the deleted elements if deleteCount is greater than 0, otherwise it returns nil.
func SpliceArray(a *ArrayAny, start Number, deleteCount Number, elements ArrayAny) ArrayAny {
	// check if the capacity is enough to store the elements, if the deleted elements are greater than or equal to the elements to be inserted,
	// then the elements to be deleted can be directly overwritten, and the remaining elements can be moved forward to the position of the elements to be deleted.
	if deleteCount >= len(elements) {
		// find the position to insert the elements, and directly overwrite the elements to be deleted
		partSlice := (*a)[:start]
		partSlice = append(partSlice, elements...)

		// move the remaining elements forward to position of the elements to be deleted.
		first := start + len(elements)
		second := start + deleteCount
		for i := second; i < len(*a); i++ {
			(*a)[first] = (*a)[second]
			first++
			second++
		}

		*a = (*a)[:first]
		return nil
	}

	// the capacity is not enough to store the elements, then the elements need to be copied
	// and the capacity is expanded to the sum of the original capacity and the length of the elements to be inserted.
	if cap(*a) < (len(*a) + len(elements) - deleteCount) {
		return SpliceArrayInner(a, start, deleteCount, elements)
	}

	// the capacity is enough to store the elements, then the elements need to be moved forward to the position of the elements to be deleted,
	// and then the elements to be inserted are appended to the slice.
	originLength := len(*a)
	(*a) = (*a)[:len(*a)+len(elements)-deleteCount]

	offset := len(elements) - deleteCount
	for i := originLength - 1; i >= start; i-- {
		(*a)[i+offset] = (*a)[i]
	}

	partSlice := (*a)[:start]
	partSlice = append(partSlice, elements...)

	return nil
}

// SpliceArrayInner copies the elements to be inserted into a new slice,
// and then appends the new slice to the original slice.
func SpliceArrayInner(a *ArrayAny, start Number, deleteCount Number, elements ArrayAny) ArrayAny {
	var deleteElements ArrayAny
	if deleteCount > 0 {
		deleteElements = (*a)[start : start+deleteCount]
	}

	combine := make(ArrayAny, 0, len(*a)+len(elements)-deleteCount)
	combine = append(combine, (*a)[:start]...)
	combine = append(combine, elements...)
	combine = append(combine, (*a)[start+deleteCount:]...)
	*a = combine

	return deleteElements
}

// CharCodeAt returns the Unicode code point of the character at the specified index in the given string.
// The index is the position of the character in the string, starting from 0.
// If the index is out of range, it returns an error.
func CharCodeAt(str string, pos Number) (uint16, error) {
	data := utf16.Encode([]rune(str))
	if pos >= len(data) || pos < 0 {
		return 0, errors.New("index out of range")
	}

	return data[pos], nil
}

// StringHeader returns the substring of the given string from the beginning to the offset.
func StringHeader(str string, offset Number) string {
	encode := utf16.Encode([]rune(str))
	if offset >= len(encode) {
		return string(utf16.Decode(encode))
	}

	return string(utf16.Decode(encode[:offset]))
}

// StringTail returns the substring of the given string from the offset to the end.
func StringTail(str string, offset Number) string {
	encode := utf16.Encode([]rune(str))
	if offset >= len(encode) {
		return ""
	}
	return string(utf16.Decode(encode[offset:]))
}

// ReplaceChar replaces the character at the specified index in the given string with the given character.
func ReplaceChar(str string, pos Number, char uint16) string {
	data := utf16.Encode([]rune(str))
	data[pos] = char
	return string(utf16.Decode(data))
}

// StringLength returns the length of the given string in utf16 code points.
func StringLength(str string) Number {
	encode := utf16.Encode([]rune(str))
	return len(encode)
}

// MergeString merges two strings.
func MergeString(str1, str2 string) string {
	// if the length of the two strings is less than or equal to 10000,
	// the fastest way is to use + to merge two strings.
	if len(str1)+len(str2) <= 10000 {
		return str1 + str2
	}

	builder := strings.Builder{}
	builder.Grow(len(str1) + len(str2))
	builder.WriteString(str1)
	builder.WriteString(str2)
	return builder.String()
}

// Conditional returns a if cond is true, otherwise b.
func Conditional(cond bool, a interface{}, b interface{}) interface{} {
	if cond {
		return a
	}

	return b
}

// GenerateNewClientID generates a new client ID.
func GenerateNewClientID() Number {
	return Number(rand.Int31())
}

// Check if `parent` is a parent of `child`.
func IsParentOf(parent IAbstractType, child *Item) bool {
	for child != nil {
		if child.Parent == parent {
			return true
		}

		child = child.Parent.(IAbstractType).GetItem()
	}

	return false
}

// Max returns the maximum value of a and b.
func Max(a, b Number) Number {
	if a > b {
		return a
	}

	return b
}

// Min returns the minimum value of a and b.
func Min(a, b Number) Number {
	if a < b {
		return a
	}

	return b
}

// ArrayLast returns the last element of the given array.
// If the array is empty, it returns an error.
func ArrayLast(a ArrayAny) (interface{}, error) {
	if len(a) == 0 {
		return nil, errors.New("empty array")
	}

	return a[len(a)-1], nil
}

// ArrayFilter filters the elements of the given array by the given filter function.
// It returns a new array that contains the elements that satisfy the filter function.
func ArrayFilter(a []IEventType, filter func(e IEventType) bool) []IEventType {
	var targets []IEventType

	for _, e := range a {
		if filter(e) {
			targets = append(targets, e)
		}
	}

	return targets
}

// Unshift adds the given element to the beginning of the given array.
func Unshift(a ArrayAny, e interface{}) ArrayAny {
	if a == nil {
		a = ArrayAny{e}
	} else {
		a = append(ArrayAny{e}, a)
	}

	return a
}

// MapAny returns true if the given map satisfies the given function.
func MapAny(m map[Number]Number, f func(key, value Number) bool) bool {
	for key, value := range m {
		if f(key, value) {
			return true
		}
	}

	return false
}

// MergeSortedRange merges two sorted ranges of the given map. The isInc parameter determines the order of the ranges.
func MapSortedRange(m map[Number]Number, isInc bool, f func(key, value Number)) {
	var keys NumberSlice
	for k := range m {
		keys = append(keys, k)
	}

	if isInc {
		sort.Sort(keys)
	} else {
		sort.Sort(sort.Reverse(keys))
	}

	for _, k := range keys {
		f(k, m[k])
	}
}

// CallAll calls all the functions in the given slice with the given arguments.
func CallAll(fs *[]func(...interface{}), args ArrayAny, i int) {
	for ; i < len(*fs); i++ {
		(*fs)[i](args...)
	}
}

// EqualAttrs returns true if the two given objects have the same attributes.
func EqualAttrs(a, b interface{}) bool {
	return reflect.DeepEqual(a, b)
}

// EqualContentFormat returns true if the two given ContentFormat objects have the same attributes.
func EqualContentFormat(a, b interface{}) bool {
	fa, ok := a.(*ContentFormat)
	if !ok {
		return false
	}

	fb, ok := b.(*ContentFormat)
	if !ok {
		return false
	}

	return EqualAttrs(fa, fb)
}

// GetUnixTime returns the current Unix time in milliseconds.
func GetUnixTime() int64 {
	return time.Now().UnixNano() / 1e6
}

// FindIndex returns the index of the first element in the given array that satisfies the given filter function.
func FindIndex(a ArrayAny, filter func(e interface{}) bool) Number {
	for i, e := range a {
		if filter(e) {
			return i
		}
	}

	return -1
}

// The top types are mapped from y.share.get(keyname) => type.
// `type` does not store any information about the `keyname`.
// This function finds the correct `keyname` for `type` and throws otherwise.
func FindRootTypeKey(t IAbstractType) string {
	// @ts-ignore _y must be defined, otherwise unexpected case
	for key, value := range t.GetDoc().Share {
		if value == t {
			return key
		}
	}

	return ""
}
