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
	Replace(v ...interface{}) int
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
	return newSet(false, slice, less, equal...)
}

// NewSafe ...
func NewSafe(s Set) Set {
	return &safeSet{
		set: s,
	}
}

func NewStable(slice interface{},
	less func(s1, s2 interface{}) bool,
	equal ...func(s1, s2 interface{}) bool,
) Set {
	return newSet(true, slice, less, equal...)
}

func NewSorted(slice interface{},
	less func(s1, s2 interface{}) bool,
	equal ...func(s1, s2 interface{}) bool,
) Set {
	return New(nil, less, equal...).New(slice, true)
}

func NewStableSorted(slice interface{},
	less func(s1, s2 interface{}) bool,
	equal ...func(s1, s2 interface{}) bool,
) Set {
	return NewStable(nil, less, equal...).New(slice, true)
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

func (p *safeSet) Replace(v ...interface{}) int {
	p.Lock()
	n := p.set.Replace(v...)
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
	swapper  func(i, j int)
	lessFunc func(slice interface{}) func(i, j int) bool
	stable   bool
}

func newSet(stable bool,
	slice interface{},
	less func(s1, s2 interface{}) bool,
	equal ...func(s1, s2 interface{}) bool) *set {
	s := &set{
		less: less,
		lessFunc: func(s interface{}) func(i, j int) bool {
			return func(i, j int) bool {
				rv := reflect.ValueOf(s)
				return less(rv.Index(i).Interface(), rv.Index(j).Interface())
			}
		},
		stable: stable,
	}
	if len(equal) > 0 {
		s.equal = equal[0]
	} else {
		s.equal = func(s1, s2 interface{}) bool {
			ok := reflect.DeepEqual(s1, s2)
			return ok
		}
	}
	if slice == nil {
		return s
	}
	s.swapper = reflect.Swapper(slice)
	rv := reflect.ValueOf(slice)
	if rv.Len() == 0 {
		s.rv = rv
	} else {
		s.rv = reflect.Zero(reflect.TypeOf(slice))
		s.InsertSlice(slice, false)
	}
	return s
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

func (p set) hasOne(v interface{}, pos int) bool {
	n := p.Search(v, pos)
	if pos+n == p.rv.Len() || !p.equal(p.rv.Index(pos+n).Interface(), v) {
		return false
	}
	return true
}

func (p set) hasSlice(slice interface{}, pos int) bool {
	p.sort(slice)
	rv := reflect.ValueOf(slice)
	if rv.Len() > p.rv.Len() {
		return false
	}
	if p.rv.Len() == 0 {
		return true
	}

	for i := 0; i < rv.Len() && pos < p.rv.Len(); i++ {
		v := rv.Index(i).Interface()
		pos += p.Search(v, pos)
		if pos == p.rv.Len() || !p.equal(p.rv.Index(pos).Interface(), v) {
			return false
		}
	}
	return true
}

func (p set) Has(v interface{}, pos int) bool {
	if reflect.TypeOf(v) == p.rv.Type() {
		return p.hasSlice(v, pos)
	}
	return p.hasOne(v, pos)
}

func (p *set) Insert(v ...interface{}) (added int) {
	for _, arg := range v {
		rv := reflect.ValueOf(arg)
		if rv.Type().Kind() == reflect.Slice {
			added += p.InsertSlice(arg, false)
			continue
		}
		added += p.InsertOne(arg)
	}
	return
}

func (p *set) Replace(v ...interface{}) (replaced int) {
	for _, arg := range v {
		rv := reflect.ValueOf(arg)
		if rv.Type().Kind() == reflect.Slice {
			replaced += p.ReplaceSlice(arg, false)
			continue
		}
		replaced += p.ReplaceOne(arg)
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
		if p.stable {
			sort.SliceStable(slice, lf)
			return
		}
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
	}

	p.rv = ReflectInsertAt(p.rv, reflect.ValueOf(v), n)
	added++
	return
}

func (p *set) ReplaceSlice(slice interface{}, sorted bool) (replaced int) {
	if !sorted {
		p.sort(slice)
	}
	if p.rv.Len() == 0 && sorted {
		p.rv = reflect.ValueOf(slice)
		replaced = p.rv.Len()
		return
	}
	rv := reflect.ValueOf(slice)
	pos := 0
	for i := 0; i < rv.Len(); i++ {
		if p.rv.Len() == 0 {
			p.rv = reflect.Append(p.rv, rv.Index(i))
			replaced++
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
				p.rv.Index(pos).Set(ri)
				continue
			} else if p.less(e, v) {
				// less than v, insert after e
				n++
			}
		} else {
			pos--
		}
		replaced++
		p.rv = ReflectInsertAt(p.rv, ri, n)
		if pos > 0 {
			pos--
		}
	}
	return
}

// ReplaceOne ...
func (p *set) ReplaceOne(v interface{}) (replaced int) {
	if p.rv.Len() == 0 {
		p.rv = reflect.Append(p.rv, reflect.ValueOf(v))
		replaced++
		return
	}
	pos := p.Search(v, 0)
	n := pos
	if pos < p.rv.Len() {
		e := p.rv.Index(pos).Interface()
		if p.equal(e, v) {
			// has v
			p.rv.Index(pos).Set(reflect.ValueOf(v))
			return
		} else if p.less(e, v) {
			// less than v, insert after e
			n++
		}
	}

	p.rv = ReflectInsertAt(p.rv, reflect.ValueOf(v), n)
	replaced++
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
	return p.new(rv, p.swapper)
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
	return p.new(dst, p.swapper)
}

func (p *set) new(rv reflect.Value, swapper func(i, j int)) *set {
	if swapper == nil && rv.IsValid() {
		swapper = reflect.Swapper(rv.Interface())
	}
	return &set{
		lessFunc: p.lessFunc,
		less:     p.less,
		equal:    p.equal,
		swapper:  swapper,
		rv:       rv,
		stable:   p.stable,
	}
}

func (p *set) Zero() Set {
	var rv reflect.Value
	if p.rv.IsValid() {
		rv = reflect.Zero(p.rv.Type())
	}
	return p.new(rv, p.swapper)
}

func (p *set) New(slice interface{}, sorted bool) Set {
	if sorted {
		return p.new(reflect.ValueOf(slice), p.swapper)
	}
	s := p.new(reflect.Zero(reflect.TypeOf(slice)), p.swapper)
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

func isSorted(sorted ...bool) bool {
	if len(sorted) == 0 {
		return false
	}
	return sorted[0]
}

// Strings ...
func Strings(arr []string, sorted ...bool) Set {
	return strings.New(arr, isSorted(sorted...))
}

// Ints ...
func Ints(arr []int, sorted ...bool) Set {
	return ints.New(arr, isSorted(sorted...))
}

// Int8s ...
func Int8s(arr []int8, sorted ...bool) Set {
	return int8s.New(arr, isSorted(sorted...))
}

// Int16s ...
func Int16s(arr []int16, sorted ...bool) Set {
	return int16s.New(arr, isSorted(sorted...))
}

// Int32s ...
func Int32s(arr []int32, sorted ...bool) Set {
	return int32s.New(arr, isSorted(sorted...))
}

// Int64s ...
func Int64s(arr []int64, sorted ...bool) Set {
	return int64s.New(arr, isSorted(sorted...))
}

// Uints ...
func Uints(arr []uint, sorted ...bool) Set {
	return uints.New(arr, isSorted(sorted...))
}

// Uint8s ...
func Uint8s(arr []uint8, sorted ...bool) Set {
	return uint8s.New(arr, isSorted(sorted...))

}

// Uint16s ...
func Uint16s(arr []uint16, sorted ...bool) Set {
	return uint16s.New(arr, isSorted(sorted...))

}

// Uint32s ...
func Uint32s(arr []uint32, sorted ...bool) Set {
	return uint32s.New(arr, isSorted(sorted...))
}

// Uint64s ...
func Uint64s(arr []uint64, sorted ...bool) Set {
	return uint64s.New(arr, isSorted(sorted...))

}

// Float32s ...
func Float32s(arr []float32, sorted ...bool) Set {
	return float32s.New(arr, isSorted(sorted...))
}

// Float64s ...
func Float64s(arr []float64, sorted ...bool) Set {
	return float64s.New(arr, isSorted(sorted...))
}

var (
	strings = NewSorted([]string{},
		func(s1, s2 interface{}) bool { return s1.(string) < s2.(string) })

	ints = NewSorted([]int{},
		func(s1, s2 interface{}) bool { return s1.(int) < s2.(int) })

	int8s = NewSorted([]int8{},
		func(s1, s2 interface{}) bool { return s1.(int8) < s2.(int8) },
	)

	int16s = NewSorted([]int16{},
		func(s1, s2 interface{}) bool { return s1.(int16) < s2.(int16) },
	)

	int32s = NewSorted([]int32{},
		func(s1, s2 interface{}) bool { return s1.(int32) < s2.(int32) },
	)

	int64s = NewSorted([]int64{},
		func(s1, s2 interface{}) bool { return s1.(int64) < s2.(int64) },
	)

	uints = NewSorted([]uint{},
		func(s1, s2 interface{}) bool { return s1.(uint) < s2.(uint) },
	)
	uint8s = NewSorted([]uint8{},
		func(s1, s2 interface{}) bool { return s1.(uint8) < s2.(uint8) },
	)
	uint16s = NewSorted([]uint16{},
		func(s1, s2 interface{}) bool { return s1.(uint16) < s2.(uint16) },
	)
	uint32s = NewSorted([]uint32{},
		func(s1, s2 interface{}) bool { return s1.(uint32) < s2.(uint32) },
	)
	uint64s = New([]uint64{},
		func(s1, s2 interface{}) bool { return s1.(uint64) < s2.(uint64) },
	)
	float32s = New([]float32{},
		func(s1, s2 interface{}) bool { return s1.(float32) < s2.(float32) },
	)
	float64s = New([]float64{},
		func(s1, s2 interface{}) bool { return s1.(float64) < s2.(float64) },
	)
)
