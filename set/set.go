package set

import (
	"reflect"
	"sort"
	"sync"
)

// Set ...
type Set interface {
	Len() int
	Slice() interface{}
	Search(v interface{}, pos int) int
	Has(v interface{}, pos int) bool
	Insert(v ...interface{}) int
	Erase(v ...interface{}) int
	ReSort()

	Equal(slice interface{}) bool
	Clone() Set
	Zero() Set
	New(slice interface{}, sorted bool) Set
	Intersection(s Set) Set
}

// New ...
func New(slice interface{},
	less func(s1, s2 interface{}) bool,
	equal ...func(s1, s2 interface{}) bool,
) Set {
	s := &set{
		less:   less,
		swaper: reflect.Swapper(slice),
		lessFunc: func(s interface{}) func(i, j int) bool {
			return func(i, j int) bool {
				rv := reflect.ValueOf(s)
				return less(rv.Index(i).Interface(), rv.Index(j).Interface())
			}
		},
	}
	if len(equal) > 0 {
		s.equal = equal[0]
	} else {
		s.equal = func(s1, s2 interface{}) bool {
			ok := reflect.DeepEqual(s1, s2)
			return ok
		}
	}
	rv := reflect.ValueOf(slice)
	if rv.Len() == 0 {
		s.rv = rv
	} else {
		s.rv = reflect.Zero(reflect.TypeOf(slice))
		s.InsertSlice(slice, false)
	}
	return s
}

// NewSafe ...
func NewSafe(s Set) Set {
	return &safeSet{
		set: s,
	}
}

// SafeSet ...
type safeSet struct {
	set Set
	sync.RWMutex
}

// var _ Set = (*safeSet)(nil)

func (p *safeSet) Len() int {
	p.RLock()
	n := p.set.Len()
	p.RUnlock()
	return n
}

func (p *safeSet) Slice() interface{} {
	p.RLock()
	s := p.set.Clone()
	p.RUnlock()
	return s.Slice()
}

func (p *safeSet) Search(v interface{}, pos int) int {
	p.RLock()
	n := p.set.Search(v, pos)
	p.RUnlock()
	return n
}

func (p *safeSet) Has(v interface{}, pos int) bool {
	p.RLock()
	ok := p.set.Has(v, pos)
	p.RUnlock()
	return ok
}

func (p *safeSet) Insert(v ...interface{}) int {
	p.Lock()
	n := p.set.Insert(v...)
	p.Unlock()
	return n
}

func (p *safeSet) Erase(v ...interface{}) int {
	p.Lock()
	n := p.set.Erase(v...)
	p.Unlock()
	return n
}

func (p *safeSet) Equal(slice interface{}) bool {
	p.RLock()
	ok := p.set.Equal(slice)
	p.RUnlock()
	return ok
}

func (p *safeSet) Clone() Set {
	p.RLock()
	s := p.set.Clone()
	p.RUnlock()
	return &safeSet{
		set: s,
	}
}

func (p *safeSet) Zero() Set {
	return &safeSet{
		set: p.set.Zero(),
	}
}

func (p *safeSet) New(slice interface{}, sorted bool) Set {
	return &safeSet{
		set: p.set.New(slice, sorted),
	}
}

func (p *safeSet) Intersection(s Set) Set {
	p.Lock()
	p.set.Intersection(s)
	p.Unlock()
	return p
}

func (p *safeSet) ReSort() {
	p.Lock()
	p.set.ReSort()
	p.Unlock()
}

// ReflectMove ...
func ReflectMove(rv reflect.Value, dstPos, srcPos, n int) {
	reflect.Copy(rv.Slice(dstPos, dstPos+n), rv.Slice(srcPos, srcPos+n))
}

// ReflectInsertAt ...
func ReflectInsertAt(slice reflect.Value, v reflect.Value, pos int) (newSlice reflect.Value) {
	newSlice = reflect.Append(slice, v)
	ReflectMove(newSlice, pos+1, pos, newSlice.Len()-(pos+1))
	newSlice.Index(pos).Set(v)
	return
}

// ReflectErase ...
func ReflectErase(slice reflect.Value, pos int) reflect.Value {
	if pos >= slice.Len() {
		return slice
	}
	if pos < slice.Len()-1 {
		ReflectMove(slice, pos, pos+1, slice.Len()-(pos+1))
	}
	return slice.Slice(0, slice.Len()-1)
}

type set struct {
	rv       reflect.Value
	less     func(s1, s2 interface{}) bool
	equal    func(s1, s2 interface{}) bool
	swaper   func(i, j int)
	lessFunc func(slice interface{}) func(i, j int) bool
}

var _ Set = (*set)(nil)

func (p set) Len() int {
	return p.rv.Len()
}

func (p set) Slice() interface{} {
	return p.rv.Interface()
}

func (p set) Search(v interface{}, pos int) int {
	return sort.Search(p.rv.Len()-pos, func(i int) bool {
		return !p.less(p.rv.Index(pos+i).Interface(), v)
	})
}

func (p set) Has(v interface{}, pos int) bool {
	n := p.Search(v, pos)
	if pos+n == p.rv.Len() || !p.equal(p.rv.Index(pos+n).Interface(), v) {
		return false
	}
	return true
}

func (p *set) Insert(v ...interface{}) (added int) {
	for _, arg := range v {
		rv := reflect.ValueOf(arg)
		if rv.Type() == p.rv.Type() {
			added += p.InsertSlice(arg, false)
			continue
		}
		added += p.InsertOne(arg)
	}
	return
}

func (p *set) Erase(v ...interface{}) (added int) {
	for _, arg := range v {
		rv := reflect.ValueOf(arg)
		if rv.Type() == p.rv.Type() {
			added += p.EraseSlice(arg, false)
			continue
		}
		added += p.EraseOne(arg)
	}
	return
}

func (p set) sort(slice interface{}) {
	lf := p.lessFunc(slice)
	if !sort.SliceIsSorted(slice, lf) {
		sort.Slice(slice, lf)
	}
}

func (p *set) InsertSlice(slice interface{}, sorted bool) (added int) {
	if !sorted {
		p.sort(slice)
	}
	if p.rv.Len() == 0 && sorted {
		p.rv = reflect.ValueOf(slice)
		added = p.rv.Len()
		return
	}
	rv := reflect.ValueOf(slice)
	pos := 0
	for i := 0; i < rv.Len(); i++ {
		if p.rv.Len() == 0 {
			p.rv = reflect.Append(p.rv, rv.Index(i))
			added++
			continue
		}
		ri := rv.Index(i)
		v := ri.Interface()
		pos += p.Search(v, pos)
		n := pos
		if pos < p.rv.Len() {
			e := p.rv.Index(pos).Interface()
			if p.equal(e, v) {
				// has v
				continue
			} else if p.less(e, v) {
				// less than v, insert after e
				n++
			}
		} else {
			pos--
		}
		added++
		p.rv = ReflectInsertAt(p.rv, ri, n)
		if pos > 0 {
			pos--
		}
	}
	return
}

func (p *set) InsertOne(v interface{}) (added int) {
	if p.rv.Len() == 0 {
		p.rv = reflect.Append(p.rv, reflect.ValueOf(v))
		added++
		return
	}
	pos := p.Search(v, 0)
	n := pos
	if pos < p.rv.Len() {
		e := p.rv.Index(pos).Interface()
		if p.equal(e, v) {
			// has v
			return
		} else if p.less(e, v) {
			// less than v, insert after e
			n++
		}
	} else {
		pos--
	}

	p.rv = ReflectInsertAt(p.rv, reflect.ValueOf(v), n)
	added++
	return
}

func (p *set) EraseOne(v interface{}) (deled int) {
	if p.rv.Len() == 0 {
		return
	}

	pos := p.Search(v, 0)
	if pos == p.rv.Len() || !p.equal(p.rv.Index(pos).Interface(), v) {
		return
	}
	p.rv = ReflectErase(p.rv, pos)
	deled = 1
	return
}

func (p *set) EraseSlice(slice interface{}, sorted bool) (deled int) {
	if p.rv.Len() == 0 {
		return
	}

	if !sorted {
		p.sort(slice)
	}
	rv := reflect.ValueOf(slice)
	pos := 0
	for i := 0; i < rv.Len() && pos < p.rv.Len(); i++ {
		v := rv.Index(i).Interface()
		pos += p.Search(v, pos)
		if pos == p.rv.Len() || !p.equal(p.rv.Index(pos).Interface(), v) {
			continue
		}
		p.rv = ReflectErase(p.rv, pos)
		deled++
	}

	return
}

func (p set) Equal(slice interface{}) bool {
	rv := reflect.ValueOf(slice)
	if p.rv.Len() != rv.Len() {
		return false
	}
	for i := 0; i < p.rv.Len(); i++ {
		if !p.equal(p.rv.Index(i).Interface(),
			rv.Index(i).Interface()) {
			return false
		}
	}
	return true
}

func (p set) Clone() Set {
	rv := reflect.MakeSlice(p.rv.Type(), p.rv.Len(), p.rv.Len())
	reflect.Copy(rv, p.rv)
	return p.new(rv)
}

func (p *set) Intersection(s Set) Set {
	pos := 0
	rv := s.(*set).rv
	dst := reflect.Zero(p.rv.Type())
	for i := 0; i < rv.Len() && pos < p.rv.Len(); i++ {
		e := rv.Index(i).Interface()
		pos += p.Search(e, pos)

		if pos == p.rv.Len() {
			continue
		}
		v := p.rv.Index(pos)
		if p.equal(v.Interface(), e) {
			dst = reflect.Append(dst, v)
		}
	}
	return p.new(dst)
}

func (p *set) new(rv reflect.Value) *set {
	return &set{
		lessFunc: p.lessFunc,
		less:     p.less,
		equal:    p.equal,
		swaper:   p.swaper,
		rv:       rv,
	}
}

func (p *set) Zero() Set {
	return p.new(reflect.Zero(p.rv.Type()))
}

func (p *set) New(slice interface{}, sorted bool) Set {
	if sorted {
		return p.new(reflect.ValueOf(slice))
	}
	s := p.Zero()
	s.Insert(slice)
	return s
}

func (p *set) SetSlice(slice interface{}) Set {
	p.rv = reflect.ValueOf(slice)
	return p
}

func (p *set) ReSort() {
	p.sort(p.Slice())
}

var (
	// Strings ...
	Strings = func(arr []string) Set {
		return New(arr,
			func(s1, s2 interface{}) bool { return s1.(string) < s2.(string) },
		)
	}
	// Ints ...
	Ints = func(arr []int) Set {
		return New(arr,
			func(s1, s2 interface{}) bool { return s1.(int) < s2.(int) },
		)
	}
	// Int8s ...
	Int8s = func(arr []int8) Set {
		return New(arr,
			func(s1, s2 interface{}) bool { return s1.(int8) < s2.(int8) },
		)
	}
	// Int16s ...
	Int16s = func(arr []int16) Set {
		return New(arr,
			func(s1, s2 interface{}) bool { return s1.(int16) < s2.(int16) },
		)
	}
	// Int32s ...
	Int32s = func(arr []int32) Set {
		return New(arr,
			func(s1, s2 interface{}) bool { return s1.(int32) < s2.(int32) },
		)
	}
	// Int64s ...
	Int64s = func(arr []int64) Set {
		return New(arr,
			func(s1, s2 interface{}) bool { return s1.(int64) < s2.(int64) },
		)
	}
	// Uints ...
	Uints = func(arr []uint) Set {
		return New(arr,
			func(s1, s2 interface{}) bool { return s1.(uint) < s2.(uint) },
		)
	}
	// Uint8s ...
	Uint8s = func(arr []uint8) Set {
		return New(arr,
			func(s1, s2 interface{}) bool { return s1.(uint8) < s2.(uint8) },
		)
	}
	// Uint16s ...
	Uint16s = func(arr []uint16) Set {
		return New(arr,
			func(s1, s2 interface{}) bool { return s1.(uint16) < s2.(uint16) },
		)
	}
	// Uint32s ...
	Uint32s = func(arr []uint32) Set {
		return New(arr,
			func(s1, s2 interface{}) bool { return s1.(uint32) < s2.(uint32) },
		)
	}
	// Uint64s ...
	Uint64s = func(arr []uint64) Set {
		return New(arr,
			func(s1, s2 interface{}) bool { return s1.(uint64) < s2.(uint64) },
		)
	}
	// Float32s ...
	Float32s = func(arr []float32) Set {
		return New(arr,
			func(s1, s2 interface{}) bool { return s1.(float32) < s2.(float32) },
		)
	}
	// Float64s ...
	Float64s = func(arr []float64) Set {
		return New(arr,
			func(s1, s2 interface{}) bool { return s1.(float64) < s2.(float64) },
		)
	}
)
