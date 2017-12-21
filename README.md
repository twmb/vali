valiface
========

[![GoDoc](https://godoc.org/github.com/twmb/valiface?status.svg)](https://godoc.org/github.com/twmb/valiface)

This package provides a function to unsafely obtain an `interface{}` value from
a `reflect.Value` without panicking. This is useful in scenarios where the
`reflect.Value` was obtained by accessing unexported struct fields.

The package is dependent on some Go internal struct layouts and constants, but
the layouts and constants have not changed in many Go releases.

Full documentation can be found on [`godoc`](https://godoc.org/github.com/twmb/valiface).
