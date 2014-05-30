
# errgo
    import "github.com/juju/errgo"

The juju/errgo provides an easy way to annotate errors without losing the
orginal error context.

The exported New and Errorf functions is designed to replace the errors.New
and fmt.Errorf functions respectively. The same underlying error is there, but
the package also records the location at which the error was created.

A primary use case for this library is to add extra context any time an
error is returned from a function.


	    if err := SomeFunc(); err != nil {
		    return err
		}

This instead becomes:


	    if err := SomeFunc(); err != nil {
		    return errgo.Trace(err)
		}

which just records the file and line number of the Trace call, or


	    if err := SomeFunc(); err != nil {
		    return errgo.Annotate(err, "more context")
		}

which also adds an annotation to the error.

Often when you want to check to see if an error is of a particular type, a
helper function is exported by the package that returned the error, like the
`os` package.  The underlying cause of the error is available using the
Cause function, or you can test the cause with the Check function.


	os.IsNotExist(errgo.Cause(err))
	
	errgo.Check(err, os.IsNotExist)

The result of the Error() call on the annotated error is the annotations
joined with colons, then the result of the Error() method
for the underlying error that was the cause.


	err := errgo.Errorf("original")
	err = errgo.Annotatef("context")
	err = errgo.Annotatef("more context")
	err.Error() -> "more context: context: original"

Obviously recording the file, line and functions is not very useful if you
cannot get them back out again.


	errgo.ErrorStack(err)

will return something like:


	first error
	github.com/juju/errgo/annotation_test.go:193:
	github.com/juju/errgo/annotation_test.go:194: annotation
	github.com/juju/errgo/annotation_test.go:195:
	github.com/juju/errgo/annotation_test.go:196: more context
	github.com/juju/errgo/annotation_test.go:197:

The first error was generated by an external system, so there was no location
associated. The second, fourth, and last lines were generated with Trace calls,
and the other two through Annotate.

If you are creating the errors, you can simply call:


	errgo.Errorf("format just like fmt.Errorf")

This function will return an error that contains the annotation stack and
records the file, line and function from the place where the error is created.

Sometimes when responding to an error you want to return a more specific error
for the situation.


	    if err := FindField(field); err != nil {
		    return errgo.Wrap(err, errors.NotFoundf(field))
		}

This returns an error where the complete error stack is still available, and
errgo.Cause will return the NotFound error.






## func Annotate
``` go
func Annotate(other error, message string) error
```
Annotate is used to add extra context to an existing error. The location of
the Annotate call is recorded with the annotations. The file, line and
function are also recorded.

For example:


	if err := SomeFunc(); err != nil {
	    return errgo.Annotate(err, "failed to frombulate")
	}


## func Annotatef
``` go
func Annotatef(other error, format string, args ...interface{}) error
```
Annotatef is used to add extra context to an existing error. The location of
the Annotate call is recorded with the annotations. The file, line and
function are also recorded.

For example:


	if err := SomeFunc(); err != nil {
	    return errgo.Annotatef(err, "failed to frombulate the %s", arg)
	}


## func Cause
``` go
func Cause(err error) error
```
Cause returns the cause of the given error.  If err does not
implement Causer or its Cause method returns nil, it returns err itself.

Cause is the usual way to diagnose errors that may have been wrapped by
the other errgo functions.


## func Check
``` go
func Check(err error, checker func(error) bool) bool
```
Check looks at the Cause of the error to see if it matches the checker
function.

For example:


	if err := SomeFunc(); err != nil {
	    if errgo.Check(err, os.IsNotExist) {
	        return someOtherFunc()
	    }
	}


## func Details
``` go
func Details(err error) string
```
Details returns information about the stack of
Previous errors wrapped by err, in the format:


	[{filename:99: error one} {otherfile:55: cause of error one}]

The details are found by type-asserting the error to
the Locationer, Causer and Wrapper interfaces.
Details of the Previous stack are found by
recursively calling Previous when the
Previous error implements Wrapper.


## func ErrorStack
``` go
func ErrorStack(err error) string
```
ErrorStack returns a string representation of the annotated error. If the
error passed as the parameter is not an annotated error, the result is
simply the result of the Error() method on that error.

If the error is an annotated error, a multi-line string is returned where
each line represents one entry in the annotation stack. The full filename
from the call stack is used in the output.


## func Errorf
``` go
func Errorf(format string, args ...interface{}) error
```
Errorf creates a new annotated error and records the location that the
error is created.  This should be a drop in replacement for fmt.Errorf.

For example:


	return errgo.Errorf("validation failed: %s", message)


## func New
``` go
func New(s string) error
```
New is a drop in replacement for the standard libary errors module that records
the location that the error is created.

For example:


	return errgo.New("validation failed")


## func Trace
``` go
func Trace(other error) error
```
Trace always returns an annotated error.  Trace records the
location of the Trace call, and adds it to the annotation stack.

For example:


	if err := SomeFunc(); err != nil {
	    return errgo.Trace(err)
	}


## func Wrap
``` go
func Wrap(other, newDescriptive error) error
```
Wrap changes the error value that is returned with LastError. The location
of the Wrap call is also stored in the annotation stack.

For example:


	if err := SomeFunc(); err != nil {
	    newErr := &packageError{"more context", private_value}
	    return errors.Wrap(err, newErr)
	}



## type Causer
``` go
type Causer interface {
    Cause() error
}
```
Causer is the type of an error that may provide
an error cause for error diagnosis. Cause may return
nil if there is no cause (for example because the
cause has been masked).











## type Err
``` go
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
```
Err holds a description of an error along with information about
where the error was created.

It may be embedded  in custom error types to add extra information that
this errors package can understand.











### func (\*Err) Cause
``` go
func (e *Err) Cause() error
```
Cause implements Causer.



### func (\*Err) Error
``` go
func (e *Err) Error() string
```
Error implements error.Error.



### func (\*Err) GoString
``` go
func (e *Err) GoString() string
```
GoString returns the details of the receiving error
message, so that printing an error with %#v will
produce useful information.



### func (\*Err) Location
``` go
func (e *Err) Location() Location
```
Location implements Locationer.



### func (\*Err) Message
``` go
func (e *Err) Message() string
```
Message returns the top level error message.



### func (\*Err) Previous
``` go
func (e *Err) Previous() error
```
Previous returns the Previous error if any.



### func (\*Err) SetLocation
``` go
func (e *Err) SetLocation(callDepth int)
```
Locate records the source location of the error by setting
e.Location, at callDepth stack frames above the call.



## type Location
``` go
type Location struct {
    File string
    Line int
}
```
Location describes a source code location.











### func (Location) IsSet
``` go
func (loc Location) IsSet() bool
```
IsSet reports whether the location has been set.



### func (Location) String
``` go
func (loc Location) String() string
```
String returns a location in filename.go:99 format.



## type Locationer
``` go
type Locationer interface {
    Location() Location
}
```
Location can be implemented by any error type
that wants to expose the source location of an error.











## type Wrapper
``` go
type Wrapper interface {
    // Message returns the top level error message,
    // not including the message from the Previous
    // error.
    Message() string

    // Previous returns the Previous error, or nil
    // if there is none.
    Previous() error
}
```
Wrapper is the type of an error that wraps another error. It is
exposed so that external types may implement it, but should in
general not be used otherwise.

















- - -
Generated by [godoc2md](http://godoc.org/github.com/davecheney/godoc2md)