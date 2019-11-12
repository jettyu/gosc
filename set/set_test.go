package set_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/jettyu/gosc/set"
)

func TestStrings(t *testing.T) {
	s := set.Strings([]string{"2", "6", "4", "5", "4", "2", "3", "0", "1"})
	arr, ok := s.Slice().([]string)
	if !ok {
		t.Fatal(s)
	}
	if len(arr) != 7 {
		t.Fatal(s.Slice(), arr)
	}
	if !s.Has("0", 0) {
		t.Fatal(s.Slice())
	}
	if s.Has("0", 1) {
		t.Fatal(s.Slice())
	}
	if !s.Has("3", 2) {
		t.Fatal(s.Slice())
	}
	if s.Has("10", 0) {
		t.Fatal(s.Slice())
	}
	if s.Insert("1", "5", "7", "8") != 2 {
		t.Fatal(s.Slice())
	}
	// 删除中间，末尾混淆
	if s.Erase("7", "9") != 1 {
		t.Fatal(s.Slice())
	}
	// 删除中间和末尾
	if s.Erase("6", "8") != 2 {
		t.Fatal(s.Slice())
	}
	// 删除开头
	if s.Erase("0", "1") != 2 {
		t.Fatal(s.Slice())
	}
	for i, v := range s.Slice().([]string) {
		if fmt.Sprint(i+2) != v {
			t.Fatal(arr)
		}
	}
	ins := s.Intersection(set.Strings([]string{"1", "2", "4", "6"}))
	if !ins.Equal([]string{"2", "4"}) {
		t.Fatal(ins.Slice())
	}
	ins.Slice().([]string)[0] = "5"
	ins.ReSort()
	if !ins.Equal([]string{"4", "5"}) {
		t.Fatal(ins.Slice())
	}
}

func TestInts(t *testing.T) {
	s := set.Ints([]int{2, 6, 4, 5, 4, 2, 3, 0, 1})
	if !s.Equal([]int{0, 1, 2, 3, 4, 5, 6}) {
		t.Fatal(s.Slice())
	}
	if !s.Has(0, 0) {
		t.Fatal(s)
	}
	if s.Has(0, 1) {
		t.Fatal(s)
	}
	if !s.Has(3, 2) {
		t.Fatal(s)
	}
	if s.Has(10, 0) {
		t.Fatal(s)
	}
	if s.Insert([]int{1, 5, 7, 8}) != 2 {
		t.Fatal(s)
	}

	if s.Erase([]int{7, 9}) != 1 {
		t.Fatal(s.Slice())
	}
	if s.Erase([]int{6, 8}) != 2 {
		t.Fatal(s)
	}
	if s.Erase([]int{0, 1}) != 2 {
		t.Fatal(s)
	}
	if !s.Equal([]int{2, 3, 4, 5}) {
		t.Fatal(s.Slice())
	}

	clone := s.Clone()
	if !s.Equal(clone.Slice()) {
		t.Fatal(clone.Slice())
	}
	s.Erase(5)
	if s.Equal(clone.Slice()) {
		t.Fatal(clone.Slice(), s.Slice())
	}
}

func TestIntersection(t *testing.T) {
	arr1 := []int{0, 1, 1, 2, 2, 4, 5}
	arr2 := []int{1, 1, 2, 2, 3, 5, 6}
	sec := set.Ints(arr1).Intersection(set.Ints(arr2))
	except := []int{1, 2, 5}

	if !set.Ints(except).Equal(sec.Slice()) {
		t.Fatal(sec.Slice())
	}
}

func TestReflectErase(t *testing.T) {
	arr := []int{0, 1, 2, 3, 4, 5}
	rv := reflect.ValueOf(arr)
	rv = set.ReflectErase(rv, 0)
	if !set.Ints(rv.Interface().([]int)).Equal([]int{1, 2, 3, 4, 5}) {
		t.Fatal(rv.Interface())
	}
	rv = set.ReflectErase(rv, 1)
	if !set.Ints(rv.Interface().([]int)).Equal([]int{1, 3, 4, 5}) {
		t.Fatal(rv.Interface())
	}
	rv = set.ReflectErase(rv, 3)
	if !set.Ints(rv.Interface().([]int)).Equal([]int{1, 3, 4}) {
		t.Fatal(rv.Interface())
	}
	rv = set.ReflectErase(rv, 3)
	if !set.Ints(rv.Interface().([]int)).Equal([]int{1, 3, 4}) {
		t.Fatal(rv.Interface())
	}
}

func TestNil(t *testing.T) {
	s := set.New(nil, func(s1, s2 interface{}) bool { return s1.(int) < s2.(int) })
	s = s.New([]int{2, 3, 1}, false)
	t.Log(s.Slice())
	s = s.New([]int{0, 1, 2}, true)
	t.Log(s.Slice())
}

type testStruct struct {
	ID    int
	Value int
}

func TestReplace(t *testing.T) {
	testStructSet := set.New([]testStruct{},
		func(s1, s2 interface{}) bool { return s1.(testStruct).ID < s2.(testStruct).ID },
		func(s1, s2 interface{}) bool { return s1.(testStruct).ID == s2.(testStruct).ID },
	)
	testStructSet.Insert([]testStruct{{1, 1}, {2, 2}, {3, 3}})
	testStructSet.Replace(testStruct{2, 5})
	i := testStructSet.Search(testStruct{ID: 2}, 0)
	if i != 1 {
		t.Fatal(testStructSet.Slice(), i)
	}
	if testStructSet.Slice().([]testStruct)[i].Value != 5 {
		t.Fatal(testStructSet.Slice())
	}
}
