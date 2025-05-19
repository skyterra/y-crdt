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
func IsUndefined(obj any) bool {
	return obj == nil || reflect.TypeOf(obj) == reflect.TypeOf(Undefined)
}

func IsNull(obj interface{}) bool {
	return reflect.TypeOf(obj) == reflect.TypeOf(Null) || (IsPtr(obj) && reflect.TypeOf(obj) != nil && reflect.ValueOf(obj).IsNil())
}

func IsPtr(obj interface{}) bool {
	return reflect.ValueOf(obj).Kind() == reflect.Ptr
}
