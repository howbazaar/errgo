package errgo_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	gc "launchpad.net/gocheck"

	"github.com/juju/errgo"
)

var (
	_ errgo.Wrapper    = (*errgo.Err)(nil)
	_ errgo.Locationer = (*errgo.Err)(nil)
	_ errgo.Causer     = (*errgo.Err)(nil)
)

type errorsSuite struct{}

var _ = gc.Suite(&errorsSuite{})

func (*errorsSuite) TestNew(c *gc.C) {
	err := errgo.New("foo") //err TestNew
	checkErr(c, err, err, "foo", "[{$TestNew$: foo}]")
}

func (*errorsSuite) TestNewf(c *gc.C) {
	err := errgo.Errorf("foo %d", 5) //err TestNewf
	checkErr(c, err, err, "foo 5", "[{$TestNewf$: foo 5}]")
}

var someErr = errgo.New("some error") //err varSomeErr

func annotate1() error {
	err := errgo.Annotate(someErr, "annotate1") //err annotate1
	return err
}

func annotate2() error {
	err := annotate1()
	err = errgo.Annotate(err, "annotate2") //err annotate2
	return err
}

func (*errorsSuite) TestAnnotateUsage(c *gc.C) {
	err0 := annotate2()
	checkErr(
		c, err0, someErr,
		"annotate2: annotate1: some error",
		"[{$annotate2$: annotate2} {$annotate1$: annotate1} {$varSomeErr$: some error}]",
	)
}

func (*errorsSuite) TestErrorString(c *gc.C) {
	for i, test := range []struct {
		message   string
		generator func() error
		expected  string
	}{
		{
			message: "uncomparable errors",
			generator: func() error {
				err := errgo.Annotatef(newNonComparableError("uncomparable"), "annotation")
				return errgo.Annotatef(err, "another")
			},
			expected: "another: annotation: uncomparable",
		}, {
			message: "Errorf",
			generator: func() error {
				return errgo.Errorf("first error")
			},
			expected: "first error",
		}, {
			message: "annotating nil",
			generator: func() error {
				return errgo.Annotatef(nil, "annotation")
			},
			expected: "annotation",
		}, {
			message: "annotated error",
			generator: func() error {
				err := errgo.Errorf("first error")
				return errgo.Annotatef(err, "annotation")
			},
			expected: "annotation: first error",
		}, {
			message: "test annotation format",
			generator: func() error {
				err := errgo.Errorf("first %s", "error")
				return errgo.Annotatef(err, "%s", "annotation")
			},
			expected: "annotation: first error",
		}, {
			message: "wrapped error",
			generator: func() error {
				err := newError("first error")
				return errgo.Wrap(err, newError("detailed error"))
			},
			expected: "detailed error",
		}, {
			message: "wrapped annotated error",
			generator: func() error {
				err := errgo.Errorf("first error")
				err = errgo.Annotatef(err, "annotated")
				return errgo.Wrap(err, fmt.Errorf("detailed error"))
			},
			expected: "detailed error",
		}, {
			message: "annotated wrapped error",
			generator: func() error {
				err := errgo.Errorf("first error")
				err = errgo.Wrap(err, fmt.Errorf("detailed error"))
				return errgo.Annotatef(err, "annotated")
			},
			expected: "annotated: detailed error",
		}, {
			message: "traced, and annotated",
			generator: func() error {
				err := errgo.New("first error")
				err = errgo.Trace(err)
				err = errgo.Annotate(err, "some context")
				err = errgo.Trace(err)
				err = errgo.Annotate(err, "more context")
				return errgo.Trace(err)
			},
			expected: "more context: some context: first error",
		},
	} {
		c.Logf("%v: %s", i, test.message)
		err := test.generator()
		ok := c.Check(err.Error(), gc.Equals, test.expected)
		if !ok {
			c.Logf("%#v", test.generator())
		}
	}
}

func (*errorsSuite) TestAnnotatedErrorCheck(c *gc.C) {
	// Look for a file that we know isn't there.
	dir := c.MkDir()
	_, err := os.Stat(filepath.Join(dir, "not-there"))
	c.Assert(os.IsNotExist(err), gc.Equals, true)
	c.Assert(errgo.Check(err, os.IsNotExist), gc.Equals, true)

	err = errgo.Annotatef(err, "wrap it")
	// Now the error itself isn't a 'IsNotExist'.
	c.Assert(os.IsNotExist(err), gc.Equals, false)
	// However if we use the Check method, it is.
	c.Assert(errgo.Check(err, os.IsNotExist), gc.Equals, true)
}

func (*errorsSuite) TestErrorStack(c *gc.C) {
	for i, test := range []struct {
		message   string
		generator func() error
		expected  string
	}{
		{
			message: "raw error",
			generator: func() error {
				return fmt.Errorf("raw")
			},
			expected: "raw",
		}, {
			message: "single error stack",
			generator: func() error {
				return errgo.New("first error") //err single
			},
			expected: "$single$: first error",
		}, {
			message: "annotated error",
			generator: func() error {
				err := errgo.New("first error")          //err annotated-0
				return errgo.Annotate(err, "annotation") //err annotated-1
			},
			expected: "" +
				"$annotated-0$: first error\n" +
				"$annotated-1$: annotation",
		}, {
			message: "wrapped error",
			generator: func() error {
				err := errgo.New("first error")                    //err wrapped-0
				return errgo.Wrap(err, newError("detailed error")) //err wrapped-1
			},
			expected: "" +
				"$wrapped-0$: first error\n" +
				"$wrapped-1$: detailed error",
		}, {
			message: "annotated wrapped error",
			generator: func() error {
				err := errgo.Errorf("first error")                  //err ann-wrap-0
				err = errgo.Wrap(err, fmt.Errorf("detailed error")) //err ann-wrap-1
				return errgo.Annotatef(err, "annotated")            //err ann-wrap-2
			},
			expected: "" +
				"$ann-wrap-0$: first error\n" +
				"$ann-wrap-1$: detailed error\n" +
				"$ann-wrap-2$: annotated",
		}, {
			message: "traced, and annotated",
			generator: func() error {
				err := errgo.New("first error")           //err stack-0
				err = errgo.Trace(err)                    //err stack-1
				err = errgo.Annotate(err, "some context") //err stack-2
				err = errgo.Trace(err)                    //err stack-3
				err = errgo.Annotate(err, "more context") //err stack-4
				return errgo.Trace(err)                   //err stack-5
			},
			expected: "" +
				"$stack-0$: first error\n" +
				"$stack-1$: \n" +
				"$stack-2$: some context\n" +
				"$stack-3$: \n" +
				"$stack-4$: more context\n" +
				"$stack-5$: ",
		}, {
			message: "uncomparable, wrapped with a value error",
			generator: func() error {
				err := newNonComparableError("first error")    //err mixed-0
				err = errgo.Trace(err)                         //err mixed-1
				err = errgo.Wrap(err, newError("value error")) //err mixed-2
				err = errgo.Trace(err)                         //err mixed-3
				err = errgo.Annotate(err, "more context")      //err mixed-4
				return errgo.Trace(err)                        //err mixed-5
			},
			expected: "" +
				"first error\n" +
				"$mixed-1$: \n" +
				"$mixed-2$: value error\n" +
				"$mixed-3$: \n" +
				"$mixed-4$: more context\n" +
				"$mixed-5$: ",
		},
	} {
		c.Logf("%v: %s", i, test.message)
		err := test.generator()
		expected := replaceLocations(test.expected)
		ok := c.Check(errgo.ErrorStack(err), gc.Equals, expected)
		if !ok {
			c.Logf("%#v", err)
		}
	}
}

type embed struct {
	*errgo.Err
}

func (*errorsSuite) TestCause(c *gc.C) {
	c.Assert(errgo.Cause(someErr), gc.Equals, someErr)

	fmtErr := fmt.Errorf("simple")
	c.Assert(errgo.Cause(fmtErr), gc.Equals, fmtErr)

	err := errgo.Wrap(someErr, fmtErr)
	c.Assert(errgo.Cause(err), gc.Equals, fmtErr)

	err = errgo.Annotate(err, "annotated")
	c.Assert(errgo.Cause(err), gc.Equals, fmtErr)

	err = &embed{err.(*errgo.Err)}
	c.Assert(errgo.Cause(err), gc.Equals, fmtErr)
}

func (*errorsSuite) TestDetails(c *gc.C) {
	c.Assert(errgo.Details(nil), gc.Equals, "[]")

	otherErr := fmt.Errorf("other")
	checkDetails(c, otherErr, "[{other}]")

	err0 := &embed{errgo.New("foo").(*errgo.Err)} //err TestStack#0
	checkDetails(c, err0, "[{$TestStack#0$: foo}]")

	err1 := &embed{errgo.Annotate(err0, "bar").(*errgo.Err)} //err TestStack#1
	checkDetails(c, err1, "[{$TestStack#1$: bar} {$TestStack#0$: foo}]")

	err2 := errgo.Trace(err1) //err TestStack#2
	checkDetails(c, err2, "[{$TestStack#2$: } {$TestStack#1$: bar} {$TestStack#0$: foo}]")
}

func (*errorsSuite) TestLocation(c *gc.C) {
	loc := errgo.Location{File: "foo", Line: 35}
	c.Assert(loc.String(), gc.Equals, "foo:35")
}

func checkDetails(c *gc.C, err error, details string) {
	c.Assert(err, gc.NotNil)
	expectedDetails := replaceLocations(details)
	c.Assert(errgo.Details(err), gc.Equals, expectedDetails)
}

func checkErr(c *gc.C, err, cause error, msg string, details string) {
	c.Assert(err, gc.NotNil)
	c.Assert(err.Error(), gc.Equals, msg)
	c.Assert(errgo.Cause(err), gc.Equals, cause)
	expectedDetails := replaceLocations(details)
	c.Assert(errgo.Details(err), gc.Equals, expectedDetails)
}

// This is an uncomparable error type, as it is a struct that supports the
// error interface (as opposed to a pointer type).
type error_ struct {
	info  string
	slice []string
}

// Create a non-comparable error
func newNonComparableError(message string) error {
	return error_{info: message}
}

func (e error_) Error() string {
	return e.info
}

func newError(message string) error {
	return testError{message}
}

// The testError is a value type error for ease of seeing results
// when the test fails.
type testError struct {
	message string
}

func (e testError) Error() string {
	return e.message
}

func replaceLocations(s string) string {
	t := ""
	for {
		i := strings.Index(s, "$")
		if i == -1 {
			break
		}
		t += s[0:i]
		s = s[i+1:]
		i = strings.Index(s, "$")
		if i == -1 {
			panic("no second $")
		}
		t += location(s[0:i]).String()
		s = s[i+1:]
	}
	t += s
	return t
}

func location(tag string) errgo.Location {
	line, ok := tagToLine[tag]
	if !ok {
		panic(fmt.Errorf("tag %q not found", tag))
	}
	return errgo.Location{
		File: "github.com/juju/errgo/errors_test.go",
		Line: line,
	}
}

var tagToLine = make(map[string]int)

func init() {
	data, err := ioutil.ReadFile("errors_test.go")
	if err != nil {
		panic(err)
	}
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if j := strings.Index(line, "//err "); j >= 0 {
			tagToLine[line[j+len("//err "):]] = i + 1
		}
	}
}
