package y_crdt

import "reflect"

type EventListener func(interface{}, interface{})

type EventHandler struct {
	L []EventListener
}

func NewEventHandler() *EventHandler {
	return &EventHandler{}
}

// Adds an event listener that is called when
func AddEventHandlerListener(eventHandler *EventHandler, f EventListener) {
	eventHandler.L = append(eventHandler.L, f)
}

// Removes an event listener.
func RemoveEventHandlerListener(eventHandler *EventHandler, f EventListener) {
	length := len(eventHandler.L)

	for i := len(eventHandler.L) - 1; i >= 0; i-- {
		if reflect.ValueOf(eventHandler.L[i]).Pointer() == reflect.ValueOf(f).Pointer() {
			eventHandler.L = append(eventHandler.L[:i], eventHandler.L[i+1:]...)
		}
	}

	if length == len(eventHandler.L) {
	}
}

// Removes all event listeners.
func RemoveAllEventHandlerListeners(eventHandler *EventHandler) {
	eventHandler.L = []EventListener{}
}

// Call all event listeners that were added via
func CallEventHandlerListeners(eventHandler *EventHandler, arg0, arg1 interface{}) {
	for _, f := range eventHandler.L {
		f(arg0, arg1)
	}
}
