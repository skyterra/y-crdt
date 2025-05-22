package y_crdt

type ObserverHandler struct {
	Once     bool
	Callback func(v ...interface{})
}

type Observable struct {
	Observers map[interface{}]Set
}

func (o *Observable) On(name interface{}, handle *ObserverHandler) {
	_, exist := o.Observers[name]
	if !exist {
		o.Observers[name] = NewSet()
	}

	o.Observers[name].Add(handle)
}

func (o *Observable) Once(name interface{}, handler *ObserverHandler) {
	handler.Once = true
	o.On(name, handler)
}

func (o *Observable) Off(name interface{}, handler *ObserverHandler) {
	observers, exist := o.Observers[name]
	if exist {
		observers.Delete(handler)
		if len(observers) == 0 {
			delete(o.Observers, name)
		}
	}
}

func (o *Observable) Emit(name interface{}, v ...interface{}) {
	observers, exist := o.Observers[name]
	if !exist {
		return
	}

	for h := range observers {
		handler, ok := h.(*ObserverHandler)
		if !ok {
			continue
		}

		if handler.Once {
			o.Off(name, handler)
		}

		handler.Callback(v...)
	}
}

func (o *Observable) Destroy() {
	o.Observers = make(map[interface{}]Set)
}

func NewObservable() *Observable {
	return &Observable{
		Observers: make(map[interface{}]Set),
	}
}

func NewObserverHandler(f func(v ...interface{})) *ObserverHandler {
	return &ObserverHandler{
		Callback: f,
	}
}
