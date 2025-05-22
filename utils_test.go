package y_crdt

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"
)

// BenchmarkMergeString benchmark the performance of MergeStringV1, MergeStringV2, MergeStringV3.
// The benchmark is run 100000 times, and the result is as follows:
//
// str length: 100
// BenchmarkMergeString/MergeStringV1 	      59.16 ns/op	     208 B/op	       1 allocs/op
// BenchmarkMergeString/MergeStringV2	      81.39 ns/op	     208 B/op	       1 allocs/op
// BenchmarkMergeString/MergeStringV3  	      170.2 ns/op	     240 B/op	       3 allocs/op
//
// str length: 1000
// BenchmarkMergeString/MergeStringV1 	      279.7 ns/op	     2048 B/op	       1 allocs/op
// BenchmarkMergeString/MergeStringV2	      287.2 ns/op	     2048 B/op	       1 allocs/op
// BenchmarkMergeString/MergeStringV3  	      360.7 ns/op	     2080 B/op	       3 allocs/op
//
// str length: 7000
// BenchmarkMergeString/MergeStringV1 	      1285 ns/op	     14336 B/op	       1 allocs/op
// BenchmarkMergeString/MergeStringV2	      1270 ns/op	     14336 B/op	       1 allocs/op
// BenchmarkMergeString/MergeStringV3  	      1597 ns/op	     14368 B/op	       3 allocs/op

func BenchmarkMergeString(b *testing.B) {
	contents := make([]string, 100000)
	for i := 0; i < 100000; i++ {
		contents[i] = randString(5000 + rand.IntN(100))
	}

	b.Run("MergeStringV1", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			MergeStringV1(contents[i%100000], contents[(i+1)%100000])
		}
	})

	b.Run("MergeStringV2", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			MergeStringV2(contents[i%100000], contents[(i+1)%100000])
		}
	})

	b.Run("MergeStringV3", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			MergeStringV3(contents[i%100000], contents[(i+1)%100000])
		}
	})
}

// user str1 + str2 to merge two strings.
func MergeStringV1(str1, str2 string) string {
	return str1 + str2
}

// use strings.Builder to merge two strings.
func MergeStringV2(str1, str2 string) string {
	builder := strings.Builder{}
	builder.Grow(len(str1) + len(str2))
	builder.WriteString(str1)
	builder.WriteString(str2)
	return builder.String()
}

func MergeStringV3(str1, str2 string) string {
	return fmt.Sprintf("%s%s", str1, str2)
}

// randString generate a random string with length n.
func randString(n int) string {
	b := make([]byte, n)
	for i := range b {
		// generate a random character between 'a' (97) and 'z' (122)
		b[i] = byte(rand.IntN(26) + 97)
	}

	return string(b)
}

func TestSpliceStruc(t *testing.T) {
	var ss []IAbstractStruct
	for i := 0; i < 10; i++ {
		ss = append(ss, &AbstractStruct{Length: Number(i)})
	}

	elements := []IAbstractStruct{&AbstractStruct{Length: 100}, &AbstractStruct{Length: 200}, &AbstractStruct{Length: 300}}
	SpliceStruct(&ss, 3, 5, elements)

	for i := 0; i < 3; i++ {
		if ss[i].GetLength() != Number(i) {
			t.Errorf("SpliceStruc(ss, 3, 5)[%d] = %d, want %d", i, ss[i].GetLength(), i)
		}
	}

	for i := 3; i < 6; i++ {
		if ss[i].GetLength() != Number(i-2)*100 {
			t.Errorf("SpliceStruc(ss, 3, 5)[%d] = %d, want %d", i, ss[i].GetLength(), (i-2)*100)
		}
	}

	for i := 6; i < 8; i++ {
		if ss[i].GetLength() != Number(i+2) {
			t.Errorf("SpliceStruc(ss, 3, 5)[%d] = %d, want %d", i, ss[i].GetLength(), i+2)
		}
	}

	ss = make([]IAbstractStruct, 0, 10)
	for i := 0; i < 10; i++ {
		ss = append(ss, &AbstractStruct{Length: Number(i)})
	}

	SpliceStruct(&ss, 3, 1, []IAbstractStruct{&AbstractStruct{Length: 100}, &AbstractStruct{Length: 200}})
	for i := 0; i < 3; i++ {
		if ss[i].GetLength() != Number(i) {
			t.Errorf("SpliceStruc(ss, 3, 1)[%d] = %d, want %d", i, ss[i].GetLength(), i)
		}
	}

	for i := 3; i < 5; i++ {
		if ss[i].GetLength() != Number(i-2)*100 {
			t.Errorf("SpliceStruc(ss, 3, 1)[%d] = %d, want %d", i, ss[i].GetLength(), (i-2)*100)
		}
	}

	for i := 5; i < 10; i++ {
		if ss[i].GetLength() != Number(i-1) {
			t.Errorf("SpliceStruc(ss, 3, 1)[%d] = %d, want %d", i, ss[i].GetLength(), i-1)
		}
	}

	ss = make([]IAbstractStruct, 0, 10)
	for i := 0; i < 8; i++ {
		ss = append(ss, &AbstractStruct{Length: Number(i)})
	}
	SpliceStruct(&ss, 3, 1, []IAbstractStruct{&AbstractStruct{Length: 100}, &AbstractStruct{Length: 200}, &AbstractStruct{Length: 300}})
	for i := 0; i < 3; i++ {
		if ss[i].GetLength() != Number(i) {
			t.Errorf("SpliceStruc(ss, 3, 1)[%d] = %d, want %d", i, ss[i].GetLength(), i)
		}
	}
	for i := 3; i < 6; i++ {
		if ss[i].GetLength() != Number(i-2)*100 {
			t.Errorf("SpliceStruc(ss, 3, 1)[%d] = %d, want %d", i, ss[i].GetLength(), (i-2)*100)
		}
	}
	for i := 6; i < 10; i++ {
		if ss[i].GetLength() != Number(i-2) {
			t.Errorf("SpliceStruc(ss, 3, 1)[%d] = %d, want %d", i, ss[i].GetLength(), i-2)
		}
	}
}
