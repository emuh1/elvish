package eval

import (
	"errors"
	"os"
)

var (
	ErrRoCannotBeSet = errors.New("read-only variable; cannot be set")
)

// Variable represents an elvish variable.
type Variable interface {
	Set(v Value)
	Get() Value
}

type ptrVariable struct {
	valuePtr  *Value
	validator func(Value) error
}

type invalidValueError struct {
	inner error
}

func (err invalidValueError) Error() string {
	return "invalid value: " + err.inner.Error()
}

func NewPtrVariable(v Value) Variable {
	return NewPtrVariableWithValidator(v, nil)
}

func NewPtrVariableWithValidator(v Value, vld func(Value) error) Variable {
	return ptrVariable{&v, vld}
}

func (iv ptrVariable) Set(val Value) {
	if iv.validator != nil {
		if err := iv.validator(val); err != nil {
			throw(invalidValueError{err})
		}
	}
	*iv.valuePtr = val
}

func (iv ptrVariable) Get() Value {
	return *iv.valuePtr
}

type roVariable struct {
	value Value
}

func NewRoVariable(v Value) Variable {
	return roVariable{v}
}

func (rv roVariable) Set(val Value) {
	throw(ErrRoCannotBeSet)
}

func (rv roVariable) Get() Value {
	return rv.value
}

type cbVariable struct {
	set func(Value)
	get func() Value
}

// MakeVariableFromCallback makes a variable from a set callback and a get
// callback.
func MakeVariableFromCallback(set func(Value), get func() Value) Variable {
	return &cbVariable{set, get}
}

func (cv *cbVariable) Set(val Value) {
	cv.set(val)
}

func (cv *cbVariable) Get() Value {
	return cv.get()
}

type roCbVariable func() Value

// MakeRoVariableFromCallback makes a read-only variable from a get callback.
func MakeRoVariableFromCallback(get func() Value) Variable {
	return roCbVariable(get)
}

func (cv roCbVariable) Set(Value) {
	throw(ErrRoCannotBeSet)
}

func (cv roCbVariable) Get() Value {
	return cv()
}

// elemVariable is a (arbitrary nested) element.
// XXX(xiaq): This is an ephemeral "variable" and is a bad hack.
type elemVariable struct {
	variable Variable
	assocers []Assocer
	indices  []Value
	setValue Value
}

var errCannotIndex = errors.New("cannot index")

func (ev *elemVariable) Set(v0 Value) {
	v := v0
	// Evaluate the actual new value from inside out. See comments in
	// compile_lvalue.go for how assignment of indexed variables work.
	for i := len(ev.assocers) - 1; i >= 0; i-- {
		v = ev.assocers[i].Assoc(ev.indices[i], v)
	}
	ev.variable.Set(v)
	// XXX(xiaq): Remember the set value for use in Get.
	ev.setValue = v0
}

func (ev *elemVariable) Get() Value {
	// XXX(xiaq): This is only called from fixNilVariables. We don't want to
	// waste time accessing the variable, so we simply return the value that was
	// set.
	return ev.setValue
}

// envVariable represents an environment variable.
type envVariable struct {
	name string
}

func (ev envVariable) Set(val Value) {
	os.Setenv(ev.name, ToString(val))
}

func (ev envVariable) Get() Value {
	return String(os.Getenv(ev.name))
}

// ErrGetBlackhole is raised when attempting to get the value of a blackhole
// variable.
var ErrGetBlackhole = errors.New("cannot get blackhole variable")

// BlackholeVariable represents a blackhole variable. Assignments to a blackhole
// variable will be discarded, and getting a blackhole variable raises an error.
type BlackholeVariable struct{}

func (bv BlackholeVariable) Set(Value) {}

func (bv BlackholeVariable) Get() Value {
	throw(ErrGetBlackhole)
	panic("unreachable")
}
