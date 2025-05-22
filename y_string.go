package y_crdt

type YString struct {
	AbstractType
	Str string
}

func (str *YString) GetLength() Number {
	return len(str.Str)
}

func (str *YString) GetItem() *Item {
	return nil
}

func (str *YString) GetMap() map[string]*Item {
	return nil
}

func (str *YString) StartItem() *Item {
	return nil
}

func (str *YString) SetStartItem(item *Item) {

}

func (str *YString) GetDoc() *Doc {
	return nil
}

func (str *YString) UpdateLength(n Number) {

}

func (str *YString) SetSearchMarker(mark []*ArraySearchMarker) {

}

func (str *YString) Parent() IAbstractType {
	return nil
}

func (str *YString) Integrate(doc *Doc, item *Item) {

}

func (str *YString) Copy() IAbstractType {
	return nil
}

func (str *YString) Clone() IAbstractType {
	return nil
}

func (str *YString) Write(encoder *UpdateEncoderV1) {

}

func (str *YString) First() *Item {
	return nil
}

func (str *YString) CallObserver(trans *Transaction, parentSubs Set) {

}

func (str *YString) Observe(f func(interface{}, interface{})) {

}

func (str *YString) ObserveDeep(f func(interface{}, interface{})) {

}

func (str *YString) Unobserve(f func(interface{}, interface{})) {

}

func (str *YString) UnobserveDeep(f func(interface{}, interface{})) {

}

func (str *YString) ToJson() interface{} {
	return ""
}

func NewYString(str string) *YString {
	ystr := &YString{
		Str: str,
	}

	ystr.EH = NewEventHandler()
	ystr.DEH = NewEventHandler()

	return ystr
}

func NewDefaultYString() *YString {
	return &YString{}
}
