package y_crdt

import "reflect"

var (
	Null      = NullType{}
	Undefined = UndefinedType{}
)

// js Number
type Number = int

// js Object
type Object = map[string]any

// js Array<any>
type ArrayAny = []any

// js undefined
type UndefinedType struct {
}

// js null
type NullType struct {
}

// js Set<any>
type Set map[any]bool

// Add adds the given element to the set.
func (s Set) Add(e any) {
	s[e] = true
}

// Has returns true if the given element is in the set.
func (s Set) Has(e any) bool {
	_, exist := s[e]
	return exist
}

// Delete deletes the given element from the set.
func (s Set) Delete(e any) {
	delete(s, e)
}

// Range calls the given function for each element in the set.
func (s Set) Range(f func(element any)) {
	for el := range s {
		f(el)
	}
}

// js NumberSlice
type NumberSlice []Number

// Len returns the length of the slice.
func (ns NumberSlice) Len() int {
	return len(ns)
}

// Less returns true if the element at index i is less than the element at index j.
func (ns NumberSlice) Less(i, j int) bool {
	return ns[i] < ns[j]
}

// Swap swaps the elements at index i and j.
func (ns NumberSlice) Swap(i, j int) {
	ns[i], ns[j] = ns[j], ns[i]
}

// Filter returns a new slice containing all elements for which the given function returns true.
func (ns NumberSlice) Filter(cond func(number Number) bool) NumberSlice {
	var r NumberSlice
	for _, n := range ns {
		if cond(n) {
			r = append(r, n)
		}
	}

	return r
}

// NewObject returns a new object.
func NewObject() Object {
	return make(Object)
}

// NewSet returns a new set.
func NewSet() Set {
	return make(Set)
}

// IsUndefined returns true if the given object is undefined.
// In javascript, undefined indicate that the variable has not been initialized.
// In golang, a nil any(=interface{}) value indicates that the variable has not been initialized.
// So, we define an object is undefined if its value is nil or its type is Undefined.
func IsUndefined(obj any) bool {
	return obj == nil || reflect.TypeOf(obj) == reflect.TypeOf(Undefined)
}

// IsNull returns true if the given object is null.
// In javascript, null indicate that the variable has been initialized and the value is null.
// In golang, we define an object is null if the object is a pointer kind and the value is nil or its type is Null.
func IsNull(obj any) bool {
	return reflect.TypeOf(obj) == reflect.TypeOf(Null) || (IsPtr(obj) && reflect.TypeOf(obj) != nil && reflect.ValueOf(obj).IsNil())
}

// IsPtr returns true if the given object is a pointer.
func IsPtr(obj any) bool {
	return reflect.ValueOf(obj).Kind() == reflect.Ptr
}

// IsGCPtr returns true if the given object is a pointer to a GC.
func IsGCPtr(obj interface{}) bool {
	return reflect.TypeOf(obj) == reflect.TypeOf(&GC{})
}

// IsItemPtr returns true if the given object is a pointer to an Item.
func IsItemPtr(obj interface{}) bool {
	return reflect.TypeOf(obj) == reflect.TypeOf(&Item{})
}

// IsIDPtr returns true if the given object is a pointer to an ID.
func IsIDPtr(obj interface{}) bool {
	return reflect.TypeOf(obj) == reflect.TypeOf(&ID{})
}

// IsSameType returns true if the given two objects are the same type.
func IsSameType(a interface{}, b interface{}) bool {
	return reflect.TypeOf(a) == reflect.TypeOf(b)
}

// IsString returns true if the given object is a string.
func IsString(obj interface{}) bool {
	return reflect.ValueOf(obj).Kind() == reflect.String
}

// IsYString returns true if the given object is a YString.
func IsYString(obj interface{}) bool {
	return reflect.TypeOf(obj) == reflect.TypeOf(&YString{})
}

// IsIAbstractType returns true if the given object is an IAbstractType.
func IsIAbstractType(a interface{}) bool {
	if a == nil {
		return false
	}

	_, ok := a.(IAbstractType)
	return ok
}
