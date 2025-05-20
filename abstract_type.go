package y_crdt

type TypeConstructor = func() IAbstractType
type IAbstractType interface {
}

type AbstractType struct {
}
