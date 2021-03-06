// Copyright 2016 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"bytes"
	"time"

	. "github.com/pingcap/check"
	"github.com/pingcap/errors"
	"github.com/pingcap/parser"
	"github.com/pingcap/parser/mysql"
	"github.com/pingcap/parser/terror"
	"github.com/pingcap/tidb/util/testleak"
)

var _ = Suite(&testMiscSuite{})

type testMiscSuite struct {
}

func (s *testMiscSuite) SetUpSuite(c *C) {
}

func (s *testMiscSuite) TearDownSuite(c *C) {
}

func (s *testMiscSuite) TestRunWithRetry(c *C) {
	defer testleak.AfterTest(c)()
	// Run succ.
	cnt := 0
	err := RunWithRetry(3, 1, func() (bool, error) {
		cnt++
		if cnt < 2 {
			return true, errors.New("err")
		}
		return true, nil
	})
	c.Assert(err, IsNil)
	c.Assert(cnt, Equals, 2)

	// Run failed.
	cnt = 0
	err = RunWithRetry(3, 1, func() (bool, error) {
		cnt++
		if cnt < 4 {
			return true, errors.New("err")
		}
		return true, nil
	})
	c.Assert(err, NotNil)
	c.Assert(cnt, Equals, 3)

	// Run failed.
	cnt = 0
	err = RunWithRetry(3, 1, func() (bool, error) {
		cnt++
		if cnt < 2 {
			return false, errors.New("err")
		}
		return true, nil
	})
	c.Assert(err, NotNil)
	c.Assert(cnt, Equals, 1)
}

func (s *testMiscSuite) TestCompatibleParseGCTime(c *C) {
	values := []string{
		"20181218-19:53:37 +0800 CST",
		"20181218-19:53:37 +0800 MST",
		"20181218-19:53:37 +0800 FOO",
		"20181218-19:53:37 +0800 +08",
		"20181218-19:53:37 +0800",
		"20181218-19:53:37 +0800 ",
		"20181218-11:53:37 +0000",
	}

	invalidValues := []string{
		"",
		" ",
		"foo",
		"20181218-11:53:37",
		"20181218-19:53:37 +0800CST",
		"20181218-19:53:37 +0800 FOO BAR",
		"20181218-19:53:37 +0800FOOOOOOO BAR",
		"20181218-19:53:37 ",
	}

	expectedTime := time.Date(2018, 12, 18, 11, 53, 37, 0, time.UTC)
	expectedTimeFormatted := "20181218-19:53:37 +0800"

	beijing, err := time.LoadLocation("Asia/Shanghai")
	c.Assert(err, IsNil)

	for _, value := range values {
		t, err := CompatibleParseGCTime(value)
		c.Assert(err, IsNil)
		c.Assert(t.Equal(expectedTime), Equals, true)

		formatted := t.In(beijing).Format(GCTimeFormat)
		c.Assert(formatted, Equals, expectedTimeFormatted)
	}

	for _, value := range invalidValues {
		_, err := CompatibleParseGCTime(value)
		c.Assert(err, NotNil)
	}
}

func (s *testMiscSuite) TestBasicFunc(c *C) {
	// Test for GetStack.
	b := GetStack()
	c.Assert(len(b) < 4096, IsTrue)

	// Test for WithRecovery.
	var recover interface{}
	WithRecovery(func() {
		panic("test")
	}, func(r interface{}) {
		recover = r
	})
	c.Assert(recover, Equals, "test")

	// Test for SyntaxError.
	c.Assert(SyntaxError(nil), IsNil)
	c.Assert(terror.ErrorEqual(SyntaxError(errors.New("test")), parser.ErrParse), IsTrue)
	c.Assert(terror.ErrorEqual(SyntaxError(parser.ErrSyntax.GenWithStackByArgs()), parser.ErrSyntax), IsTrue)

	// Test for SyntaxWarn.
	c.Assert(SyntaxWarn(nil), IsNil)
	c.Assert(terror.ErrorEqual(SyntaxWarn(errors.New("test")), parser.ErrParse), IsTrue)

	// Test for ProcessInfo.
	pi := ProcessInfo{
		ID:      1,
		User:    "test",
		Host:    "www",
		DB:      "db",
		Command: mysql.ComSleep,
		Plan:    nil,
		Time:    time.Now(),
		State:   1,
		Info:    "test",
	}
	row := pi.ToRow(false)
	row2 := pi.ToRow(true)
	c.Assert(row, DeepEquals, row2)
	c.Assert(len(row), Equals, 8)
	c.Assert(row[0], Equals, pi.ID)
	c.Assert(row[1], Equals, pi.User)
	c.Assert(row[2], Equals, pi.Host)
	c.Assert(row[3], Equals, pi.DB)
	c.Assert(row[4], Equals, "Sleep")
	c.Assert(row[5], Equals, uint64(0))
	c.Assert(row[6], Equals, "1")
	c.Assert(row[7], Equals, "test")

	// Test for RandomBuf.
	buf := RandomBuf(5)
	c.Assert(len(buf), Equals, 5)
	c.Assert(bytes.Contains(buf, []byte("$")), IsFalse)
	c.Assert(bytes.Contains(buf, []byte{0}), IsFalse)
}
