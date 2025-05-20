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

type Set map[any]bool

func (s Set) Add(e any) {
	s[e] = true
}

func (s Set) Has(e any) bool {
	_, exist := s[e]
	return exist
}

func (s Set) Delete(e any) {
	delete(s, e)
}

func (s Set) Range(f func(element any)) {
	for el := range s {
		f(el)
	}
}

type NumberSlice []Number

func (ns NumberSlice) Len() int {
	return len(ns)
}

func (ns NumberSlice) Less(i, j int) bool {
	return ns[i] < ns[j]
}

func (ns NumberSlice) Swap(i, j int) {
	ns[i], ns[j] = ns[j], ns[i]
}

func (ns NumberSlice) Filter(cond func(number Number) bool) NumberSlice {
	var r NumberSlice
	for _, n := range ns {
		if cond(n) {
			r = append(r, n)
		}
	}

	return r
}

func NewObject() Object {
	return make(Object)
}

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
