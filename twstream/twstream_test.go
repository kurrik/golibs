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
	"github.com/kurrik/golibs/twurlrc"
	"net/url"
	"strings"
)

type MockDialer struct {
	t *testing.T
	Conn *MockConnection
}

func NewMockDialer(t *testing.T) *MockDialer {
	return &MockDialer{Conn: &MockConnection{t: t}}
}

func (d *MockDialer) Dial(addr string) (io.ReadWriteCloser, error) {
	return d.Conn, nil
}

const (
	READ int = iota
	WRITE
	CLOSE
	EMPTY
	EOF
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
	c.t.Log("getExpected")
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
	c.t.Log("read")
	command, message := c.getExpected()
	if command == EOF {
		c.t.Log("Sending EOF")
		return 0, io.EOF
	}
	if command != READ {
		c.t.Fatal("Unexpected READ")
	}
	copy(p, []byte(message))
	return len(message), nil
}

func (c *MockConnection) Write(p []byte) (n int, err error) {
	c.t.Log("write")
	command, message := c.getExpected()
	if command != WRITE {
		c.t.Fatal("Unexpected WRITE")
	}
	if message != string(p) {
		c.t.Errorf("Expected '%v', got '%v'", []byte(message), p)
	}
	return len(p), nil
}

func (c *MockConnection) Close() error {
	c.t.Log("close")
	command, _ := c.getExpected()
	if command != CLOSE {
		c.t.Fatal("Unexpected CLOSE")
	}
	return nil
}

func (c *MockConnection) EndTest() {
	if len(c.commands) > 0 {
		c.t.Error("MockConnection commands still in queue")
	}
}

var (
	CRLF = string([]byte{13, 10})
	CONNECT_STRING = strings.Join([]string{
		"GET /1/statuses/filter.json HTTP/1.1",
		"Host: stream.twitter.com",
		"User-Agent: Go http package",
		"Authorization: OAuth " +
			"oauth_consumer_key=\"consumerkey\", " +
			"oauth_nonce=\"54321\", " +
			"oauth_signature=\"dG59sMu9QpDU4oJMGCjKEKGlVYU%3D\", " +
			"oauth_signature_method=\"HMAC-SHA1\", " +
			"oauth_timestamp=\"12345\", " +
			"oauth_token=\"token\", " +
			"oauth_version=\"1.0\"",
		"Connection: close",
		CRLF,
	}, CRLF)
	PAYLOAD_STRING_1 = "{\"foo\": \"bar\"}" + CRLF
)

func TestParse(t *testing.T) {
	dialer := NewMockDialer(t)
	dialer.Conn.Expect(WRITE, CONNECT_STRING)
	dialer.Conn.Expect(READ, PAYLOAD_STRING_1)
	dialer.Conn.Expect(EOF, "")
	dialer.Conn.Expect(CLOSE, "")
	defer dialer.Conn.EndTest()

	requestUrl, _ := url.Parse("https://stream.twitter.com/1/statuses/filter.json")
	conf := &Configuration{
		Method: "GET",
		URL: requestUrl,
		Chunked: false,
		GZip: false,
	}
	cred := &twurlrc.Credentials{
		Token: "token",
		Username: "username",
		ConsumerKey: "consumerkey",
		ConsumerSecret: "consumersecret",
		Secret: "secret",
	}
	conn := NewConnection(conf, cred)
	conn.fixedTime = "12345"
	conn.fixedNonce = "54321"
	conn.dialer = dialer
	conn.Read()
}
