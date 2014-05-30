package errgo

import (
	"bytes"
	"fmt"
	"reflect"
	"runtime"
	"strings"
)

const debug = false

// Location describes a source code location.
type Location struct {
	File string
	Line int
}

// String returns a location in filename.go:99 format.
func (loc Location) String() string {
	return fmt.Sprintf("%s:%d", loc.File, loc.Line)
}

// IsSet reports whether the location has been set.
func (loc Location) IsSet() bool {
	return loc.File != ""
}

// Err holds a description of an error along with information about
// where the error was created.
//
// It may be embedded  in custom error types to add extra information that
// this errors package can understand.
type Err struct {
	// Message_ holds an annotation of the error.
	Message_ string

	// Cause_ holds the cause of the error as returned
	// by the Cause method.
	Cause_ error

	// Previous holds the Previous error in the error stack, if any.
	Previous_ error

	// Location holds the source code location where the error was
	// created.
	Location_ Location
}

// Location implements Locationer.
func (e *Err) Location() Location {
	return e.Location_
}

// Previous returns the Previous error if any.
func (e *Err) Previous() error {
	return e.Previous_
}

// Cause implements Causer.
func (e *Err) Cause() error {
	return e.Cause_
}

// Message returns the top level error message.
func (e *Err) Message() string {
	return e.Message_
}

// Error implements error.Error.
func (e *Err) Error() string {
	// We want to walk up the stack of errors showing the annotations
	// as long as the cause is the same.
	err := e.Previous_
	if !sameError(Cause(err), e.Cause_) && e.Cause_ != nil {
		err = e.Cause_
	}
	switch {
	case err == nil:
		return e.Message_
	case e.Message_ == "":
		return err.Error()
	}
	return fmt.Sprintf("%s: %v", e.Message_, err)
}

// GoString returns the details of the receiving error
// message, so that printing an error with %#v will
// produce useful information.
func (e *Err) GoString() string {
	return Details(e)
}

// Causer is the type of an error that may provide
// an error cause for error diagnosis. Cause may return
// nil if there is no cause (for example because the
// cause has been masked).
type Causer interface {
	Cause() error
}

// Wrapper is the type of an error that wraps another error. It is
// exposed so that external types may implement it, but should in
// general not be used otherwise.
type Wrapper interface {
	// Message returns the top level error message,
	// not including the message from the Previous
	// error.
	Message() string

	// Previous returns the Previous error, or nil
	// if there is none.
	Previous() error
}

// Location can be implemented by any error type
// that wants to expose the source location of an error.
type Locationer interface {
	Location() Location
}

// Details returns information about the stack of
// Previous errors wrapped by err, in the format:
//
// 	[{filename:99: error one} {otherfile:55: cause of error one}]
//
// The details are found by type-asserting the error to
// the Locationer, Causer and Wrapper interfaces.
// Details of the Previous stack are found by
// recursively calling Previous when the
// Previous error implements Wrapper.
func Details(err error) string {
	if err == nil {
		return "[]"
	}
	var s []byte
	s = append(s, '[')
	for {
		s = append(s, '{')
		if err, ok := err.(Locationer); ok {
			loc := err.Location()
			if loc.IsSet() {
				s = append(s, loc.String()...)
				s = append(s, ": "...)
			}
		}
		if cerr, ok := err.(Wrapper); ok {
			s = append(s, cerr.Message()...)
			err = cerr.Previous()
		} else {
			s = append(s, err.Error()...)
			err = nil
		}
		if debug {
			if err, ok := err.(Causer); ok {
				if cause := err.Cause(); cause != nil {
					s = append(s, fmt.Sprintf("=%T", cause)...)
					s = append(s, Details(cause)...)
				}
			}
		}
		s = append(s, '}')
		if err == nil {
			break
		}
		s = append(s, ' ')
	}
	s = append(s, ']')
	return string(s)
}

// Locate records the source location of the error by setting
// e.Location, at callDepth stack frames above the call.
func (e *Err) SetLocation(callDepth int) {
	_, file, line, _ := runtime.Caller(callDepth + 1)
	e.Location_ = Location{trimGoPath(file), line}
}

func setLocation(err error, callDepth int) {
	if e, _ := err.(*Err); e != nil {
		e.SetLocation(callDepth + 1)
	}
}

// New is a drop in replacement for the standard libary errors module that records
// the location that the error is created.
//
// For example:
//    return errgo.New("validation failed")
//
func New(s string) error {
	err := &Err{Message_: s}
	err.SetLocation(1)
	return err
}

// Errorf creates a new annotated error and records the location that the
// error is created.  This should be a drop in replacement for fmt.Errorf.
//
// For example:
//    return errgo.Errorf("validation failed: %s", message)
//
func Errorf(format string, args ...interface{}) error {
	err := &Err{Message_: fmt.Sprintf(format, args...)}
	err.SetLocation(1)
	return err
}

// Trace always returns an annotated error.  Trace records the
// location of the Trace call, and adds it to the annotation stack.
//
// For example:
//   if err := SomeFunc(); err != nil {
//       return errgo.Trace(err)
//   }
//
func Trace(other error) error {
	err := &Err{Previous_: other, Cause_: Cause(other)}
	err.SetLocation(1)
	return err
}

// Annotate is used to add extra context to an existing error. The location of
// the Annotate call is recorded with the annotations. The file, line and
// function are also recorded.
//
// For example:
//   if err := SomeFunc(); err != nil {
//       return errgo.Annotate(err, "failed to frombulate")
//   }
//
func Annotate(other error, message string) error {
	// Underlying is the previous link used for traversing the stack.
	// Cause is the reason for this error.
	err := &Err{
		Previous_: other,
		Cause_:    Cause(other),
		Message_:  message,
	}
	err.SetLocation(1)
	return err
}

// Annotatef is used to add extra context to an existing error. The location of
// the Annotate call is recorded with the annotations. The file, line and
// function are also recorded.
//
// For example:
//   if err := SomeFunc(); err != nil {
//       return errgo.Annotatef(err, "failed to frombulate the %s", arg)
//   }
//
func Annotatef(other error, format string, args ...interface{}) error {
	// Underlying is the previous link used for traversing the stack.
	// Cause is the reason for this error.
	err := &Err{
		Previous_: other,
		Cause_:    Cause(other),
		Message_:  fmt.Sprintf(format, args...),
	}
	err.SetLocation(1)
	return err
}

// Wrap changes the error value that is returned with LastError. The location
// of the Wrap call is also stored in the annotation stack.
//
// For example:
//   if err := SomeFunc(); err != nil {
//       newErr := &packageError{"more context", private_value}
//       return errors.Wrap(err, newErr)
//   }
//
func Wrap(other, newDescriptive error) error {
	err := &Err{
		Previous_: other,
		Cause_:    newDescriptive,
	}
	err.SetLocation(1)
	return err
}

// Mask masks the given error with the given format string and arguments (like
// fmt.Sprintf), returning a new error that maintains the error stack, but
// hides the underlying error type.  The error string still contains the full
// annotations. If you want to hide the annotatinos, call Wrap.
func Maskf(other error, format string, args ...interface{}) error {
	err := &Err{
		Message_:  fmt.Sprintf(format, args...),
		Previous_: other,
	}
	err.SetLocation(1)
	return err
}

// Mask is a simpler version of Maskf that takes no formatting arguments.
func Mask(other error, message string) error {
	err := &Err{
		Message_:  message,
		Previous_: other,
	}
	err.SetLocation(1)
	return err
}

// Check looks at the Cause of the error to see if it matches the checker
// function.
//
// For example:
//   if err := SomeFunc(); err != nil {
//       if errgo.Check(err, os.IsNotExist) {
//           return someOtherFunc()
//       }
//   }
//
func Check(err error, checker func(error) bool) bool {
	return checker(Cause(err))
}

// ErrorStack returns a string representation of the annotated error. If the
// error passed as the parameter is not an annotated error, the result is
// simply the result of the Error() method on that error.
//
// If the error is an annotated error, a multi-line string is returned where
// each line represents one entry in the annotation stack. The full filename
// from the call stack is used in the output.
func ErrorStack(err error) string {
	if err == nil {
		return ""
	}
	// We want the first error first
	var lines []string
	for {
		var buff []byte
		if err, ok := err.(Locationer); ok {
			loc := err.Location()
			// Strip off the leading GOPATH/src path elements.
			loc.File = trimGoPath(loc.File)
			if loc.IsSet() {
				buff = append(buff, loc.String()...)
				buff = append(buff, ": "...)
			}
		}
		if cerr, ok := err.(Wrapper); ok {
			message := cerr.Message()
			buff = append(buff, message...)
			// If there is a cause for this error, and it is different to the cause
			// of the underlying error, then output the error string in the stack trace.
			var cause error
			if err1, ok := err.(Causer); ok {
				cause = err1.Cause()
			}
			err = cerr.Previous()
			if cause != nil && !sameError(Cause(err), cause) {
				if message != "" {
					buff = append(buff, ": "...)
				}
				buff = append(buff, cause.Error()...)
			}
		} else {
			buff = append(buff, err.Error()...)
			err = nil
		}
		lines = append(lines, string(buff))
		if err == nil {
			break
		}
	}
	// reverse the lines to get the original error, which was at the end of
	// the list, back to the start.
	var result []string
	for i := len(lines); i > 0; i-- {
		result = append(result, lines[i-1])
	}
	return strings.Join(result, "\n")
}

// Ideally we'd have a way to check identity, but deep equals will do.
func sameError(e1, e2 error) bool {
	return reflect.DeepEqual(e1, e2)
}

// Cause returns the cause of the given error.  If err does not
// implement Causer or its Cause method returns nil, it returns err itself.
//
// Cause is the usual way to diagnose errors that may have been wrapped by
// the other errgo functions.
func Cause(err error) error {
	if err, ok := err.(Causer); ok {
		if diag := err.Cause(); diag != nil {
			return diag
		}
	}
	return err
}

// callers returns the stack trace of the goroutine that called it,
// starting n entries above the caller of callers, as a space-separated list
// of filename:line-number pairs with no new lines.
func callers(n, max int) []byte {
	var b bytes.Buffer
	prev := false
	for i := 0; i < max; i++ {
		_, file, line, ok := runtime.Caller(n + 1)
		if !ok {
			return b.Bytes()
		}
		if prev {
			fmt.Fprintf(&b, " ")
		}
		fmt.Fprintf(&b, "%s:%d", file, line)
		n++
		prev = true
	}
	return b.Bytes()
}
