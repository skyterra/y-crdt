package y_crdt

import (
	"encoding/json"
	"time"
)

func ClearInterval(t *time.Timer) {
	t.Stop()
}

func JsonString(object interface{}) string {
	data, err := json.Marshal(object)
	if err != nil {
		return ""
	}

	return string(data)
}

func JsonObject(data string) interface{} {
	var object interface{}
	err := json.Unmarshal([]byte(data), &object)
	if err != nil {
		// todo trace error
		return nil
	}

	return object
}

func AwarenessStatesKeys(states map[Number]Object) []Number {
	v := make([]Number, 0, len(states))
	for k, _ := range states {
		v = append(v, k)
	}
	return v
}
