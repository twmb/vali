// Package valiface provides a function to obtain an interface{} from a
// reflect.Value without panicking, even if the value was obtained by accessing
// unexported fields.
//
// This package relies on Go internal struct layouts and flags, but the parts
// this relies on have not changed at least two years. As with all code of this
// kind, exercise caution before using it in production.
package valiface

import (
	"reflect"
	"unsafe"
)

const (
	unknownLayout = "reflect.Value known layout changed"

	// What follows are the constants that have not changed for a long
	// time. kind is for reflect.Value's rtype's kind uint8; flag is for
	// reflect.Value's flag uintptr.
	kindDirectIface = 1 << 5
	flagIndir       = 1 << 7
	flagAddr        = 1 << 8
)

// We save some field offsets so that we can access reflect.Value directly.
var typOffset, ptrOffset, typKindOffset, flagOffset uintptr

func init() {
	var v reflect.Value
	t := reflect.TypeOf(v)
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		switch sf.Name {
		case "typ":
			typOffset = sf.Offset

			kt, found := sf.Type.Elem().FieldByName("kind")
			if !found {
				panic(unknownLayout)
			}
			typKindOffset = kt.Offset + typOffset

		case "ptr":
			ptrOffset = sf.Offset
		case "flag":
			flagOffset = sf.Offset
		default:
			panic(unknownLayout)
		}
	}
}

type ifaceWords struct {
	typ, data unsafe.Pointer
}

// Interface returns an interface of the value inside v.
//
// This function returns nil in the following two cases: (1) the value contains
// an invalid kind (likely, the value is nil itself), or (2) the value was
// obtained from CanAddr. The latter scenario requires hooking into the runtime
// to allocate a new object, something that this code cannot do.
func Interface(v reflect.Value) interface{} {
	// If the value pointer is an interface, we can return that directly.
	ptr := unsafe.Pointer(uintptr(unsafe.Pointer(&v)) + ptrOffset)
	if v.Kind() == reflect.Interface {
		return *(*interface{})(ptr)
	}

	// We cannot pack invalid values. This would be from, for example,
	// Interface() on a nil interface.
	typPtr := (*unsafe.Pointer)(unsafe.Pointer(uintptr(unsafe.Pointer(&v)) + typOffset))
	if typPtr == nil {
		// no type associated, kind is Invalid
		return nil
	}

	// reflect/value.go's packEface follows below.
	typ := *typPtr
	typKind := *(*uint8)(unsafe.Pointer(uintptr(typ) + typKindOffset))
	flag := *(*uintptr)(unsafe.Pointer(uintptr(unsafe.Pointer(&v)) + flagOffset))

	var i interface{}
	iw := (*ifaceWords)(unsafe.Pointer(&i))

	// The logic below is the same as in reflect.Value, but I do not
	// understand the Go internals _enough_ to decipher what exactly
	// kindDirectIface is. It appears to be most things behind a pointer.
	//
	// I don't think the last else statement is used anymore - Go used to
	// embed values directly into interfaces if they could fit, but they
	// removed that around 1.5 or so.
	if typKind&kindDirectIface == 0 {
		if flag&flagIndir == 0 {
			// Not much we can do here, and even reflect.Value
			// panics, so we will keep this.
			panic("bad indir")
		}
		if flag&flagAddr != 0 {
			// we cannot malloc a new type here to fill in behind
			// a pointer... see reflect/value.go
			return nil
		}
		iw.data = ptr
	} else if flag&flagIndir != 0 {
		iw.data = *(*unsafe.Pointer)(ptr)
	} else {
		iw.data = ptr
	}

	// We fill in "typ" last so that the GC does not observe a partially
	// built interface value.
	iw.typ = typ
	return i
}
