// Copyright 2012 Twitter, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package twstream

import (
	"testing"
	"io"
	"fmt"
)

type MockDialer struct {
	t *testing.T
}

func (d *MockDialer) Dial(addr string) (io.ReadWriteCloser, error) {
	return &MockConnection{t: d.t}, nil
}

const (
	READ int = iota
	WRITE
	CLOSE
	EMPTY
)

type MockConnection struct {
	messages []string
	commands []int
	t        *testing.T
}

func (c *MockConnection) Expect(command int, message string) {
	c.messages = append(c.messages, message)
	c.commands = append(c.commands, command)
}

func (c *MockConnection) getExpected() (int, string) {
	if len(c.messages) == 0 {
		return EMPTY, ""
	}
	message := c.messages[0]
	command := c.commands[0]
	c.messages = append(c.messages[:0], c.messages[1:]...)
	c.commands = append(c.commands[:0], c.commands[1:]...)
	return command, message
}

func (c *MockConnection) Read(p []byte) (n int, err error) {
	command, message := c.getExpected()
	if command != READ {
		c.t.Error("Expected READ")
	}
	p = append(p, []byte(message)...)
	return len(message), nil
}

func (c *MockConnection) Write(p []byte) (n int, err error) {
	command, message := c.getExpected()
	if command != WRITE {
		c.t.Error("Expected WRITE")
	}
	if message != string(p) {
		c.t.Error(fmt.Sprintf("Expected '%v', got '%v'", message, p))
	}
	return len(p), nil
}

func (c *MockConnection) Close() error {
	command, _ := c.getExpected()
	if command != CLOSE {
		c.t.Error("Expected CLOSE")
	}
	return nil
}

func (c *MockConnection) EndTest() {
	if len(c.commands) > 0 {
		c.t.Error("MockConnection commands still in queue")
	}
}

func TestParse(t *testing.T) {
	t.Error("Fail")
}
