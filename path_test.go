// Copyright 2013, 2014 Canonical Ltd.
// Licensed under the GPLv3, see LICENCE file for details.

package errgo_test

import (
	"path"

	gc "launchpad.net/gocheck"

	"github.com/juju/errgo"
)

type pathSuite struct{}

var _ = gc.Suite(&pathSuite{})

func (*pathSuite) TestGoPathSet(c *gc.C) {
	c.Assert(errgo.GoPath(), gc.Not(gc.Equals), "")
}

func (*pathSuite) TestTrimGoPath(c *gc.C) {
	relativeImport := "github.com/foo/bar/baz.go"
	filename := path.Join(errgo.GoPath(), relativeImport)
	c.Assert(errgo.TrimGoPath(filename), gc.Equals, relativeImport)

	absoluteImport := "/usr/share/foo/bar/baz.go"
	c.Assert(errgo.TrimGoPath(absoluteImport), gc.Equals, absoluteImport)
}
