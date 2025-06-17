package y_crdt

import (
	"testing"
)

func TestTextInsertDelete(t *testing.T) {
	// Generated via:
	//     ```js
	//        const doc = new Y.Doc()
	//        const ytext = doc.getText('type')
	//        doc..transact_mut()(function () {
	//            ytext.insert(0, 'def')
	//            ytext.insert(0, 'abc')
	//            ytext.insert(6, 'ghi')
	//            ytext.delete(2, 5)
	//        })
	//        const update = Y.encodeStateAsUpdate(doc)
	//        ytext.toString() // => 'abhi'
	//     ```
	//
	//     This way we confirm that we can decode and apply:
	//     1. blocks without left/right origin consisting of multiple characters
	//     2. blocks with left/right origin consisting of multiple characters
	//     3. delete sets

	// construct doc by golang and check to see if the result is the same as the expected.
	doc := NewDoc("guid", false, nil, nil, false)
	ytext := doc.GetText("type")
	doc.Transact(func(trans *Transaction) {
		ytext.Insert(0, "def", nil)
		ytext.Insert(0, "abc", nil)
		ytext.Insert(6, "ghi", nil)
		ytext.Delete(2, 5)
	}, nil)

	if ytext.ToString() != "abhi" {
		t.Error("expected abhi, got ", ytext.ToString())
	}
	t.Logf("construct by golang, ytext is %s", ytext.ToString())

	// apply the update and check to see if the result is the same as the expected.
	var jsUpdate = []byte{
		1, 5, 152, 234, 173, 126, 0, 1, 1, 4, 116, 121, 112, 101, 3, 68, 152, 234, 173, 126, 0, 2,
		97, 98, 193, 152, 234, 173, 126, 4, 152, 234, 173, 126, 0, 1, 129, 152, 234, 173, 126, 2,
		1, 132, 152, 234, 173, 126, 6, 2, 104, 105, 1, 152, 234, 173, 126, 2, 0, 3, 5, 2,
	}

	doc = NewDoc("guid", false, nil, nil, false)
	doc.Transact(func(trans *Transaction) {
		ApplyUpdate(doc, jsUpdate, nil)
	}, nil)

	ytext = doc.GetText("type")
	if ytext.ToString() != "abhi" {
		t.Errorf("expected abhi, got %s", ytext.ToString())
	}
	t.Logf("after apply update, ytext is %s", ytext.ToString())
}
