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
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/kurrik/golibs/oauth1a"
	"github.com/kurrik/golibs/twurlrc"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Configuration struct {
	Method         string
	URL            *url.URL
	Chunked        bool
	Proxy          string
	WriterListener io.Writer
	ReaderListener io.Writer
	TTL            int64
	GZip           bool
}

// Returns an integer representation of a hex string encoded as a series of
// ASCII bytes.
func decodeHexString(data []byte) (uint64, error) {
	var size uint64 = 0
	var i uint8
	for _, c := range data {
		switch {
		case '0' <= c && c <= '9':
			i = c - '0'
		case 'a' <= c && c <= 'f':
			i = c - 'a' + 10
		case 'A' <= c && c <= 'F':
			i = c - 'A' + 10
		default:
			return 0, errors.New("Invalid hex")
		}
		size = size*16 + uint64(i)
	}
	return size, nil
}

type listeningReader struct {
	reader   io.Reader
	listener io.Writer
}

func (r *listeningReader) Read(p []byte) (n int, err error) {
	n, err = r.Read(p)
	if err == nil {
		n, err = r.listener.Write(p)
	}
	return n, err
}

// A wrapper around an io.Writer which will only write non-empty or non-\r\n
// responses.
type NonEmptyWriter struct {
	Writer io.Writer
}

// Write p into the configured writer if len(p) > 0 and p != "\r\n".
// Returns len(p) if nothing is written, or the number of bytes actually written
// and any errors which may have occurred.
func (w *NonEmptyWriter) Write(p []byte) (n int, err error) {
	size := len(p)
	if size == 0 || size == 2 && string(p) == "\r\n" {
		return size, nil
	}
	return w.Writer.Write(p)
}

type Connection struct {
	conf   *Configuration
	cred   *twurlrc.Credentials
	conn   net.Conn
	writer io.Writer
	reader *bufio.Reader
}

func NewConnection(conf *Configuration, cred *twurlrc.Credentials) *Connection {
	return &Connection{conf: conf, cred: cred}
}

func (c *Connection) Read() error {
	err := c.connect()
	if err != nil {
		return err
	}
	defer c.conn.Close()
	if c.conf.WriterListener != nil {
		c.writer = io.MultiWriter(c.conn, c.conf.WriterListener)
	} else {
		c.writer = c.conn
	}
	if c.conf.ReaderListener != nil {
		c.reader = bufio.NewReader(&listeningReader{
			reader:   c.conn,
			listener: c.conf.ReaderListener,
		})
	} else {
		c.reader = bufio.NewReader(c.conn)
	}
	c.request()
	err = c.readHeaders()
	if err != nil {
		return err
	}
	if c.conf.Chunked {
		err = c.readChunkedData()
	} else {
		err = c.readData()
	}
	return err
}

// Reads a stream until the first blank line is found.
// Used to ignore a HTTP header response on an input stream.
func (c *Connection) readHeaders() error {
	var line []byte
	var err error
	var isGZip bool = false
	for {
		line, _, err = c.reader.ReadLine()
		lowerLine := strings.ToLower(string(line))
		if strings.HasPrefix(lowerLine, "content-encoding:") {
			if strings.Index(lowerLine, "gzip") > -1 {
				isGZip = true
			}
		}
		if string(line) == "" {
			break
		}
		if err != nil {
			return err
		}
	}
	if c.conf.GZip == true && isGZip == false {
		c.conf.GZip = false
	}
	return nil
}

// Reads non-chunked lines from the connection reader.
func (c *Connection) readData() error {
	var err error
	var line []byte
	var start time.Time

	if c.conf.GZip == true {
		z, err := gzip.NewReader(c.reader)
		if err != nil {
			return err
		}
		defer z.Close()
		c.reader = bufio.NewReader(z)
	}

	start = time.Now()
	for err == nil {
		line, _, err = c.reader.ReadLine()
		if err != nil {
			return err
		}
		fmt.Println(string(line))
		if c.conf.TTL > 0 {
			if time.Now().Sub(start).Nanoseconds() > c.conf.TTL {
				return nil
			}
		}
	}
	return err
}

// Reads transfer-encoding: chunked payloads from the connection reader.
func (c *Connection) readChunkedData() error {
	var err error
	var line []byte
	var size uint64
	var start time.Time

	start = time.Now()
	writer := &NonEmptyWriter{os.Stdout}

	var buffer *bytes.Buffer
	var decompressor *gzip.Reader
	var zipReader *bufio.Reader
	var data []byte

	if c.conf.GZip == true {
		buffer = bytes.NewBufferString("")
	}

	for err == nil {
		line, _, err = c.reader.ReadLine()
		if err != nil {
			return err
		}
		size, err = decodeHexString(line)
		if err != nil {
			str := fmt.Sprintf("Expected hex, got %v", string(line))
			return errors.New(str)
		}
		if c.conf.GZip == false {
			_, err = io.CopyN(writer, c.reader, int64(size))
		} else {
			_, err = io.CopyN(buffer, c.reader, int64(size))
			if err != nil {
				return err
			}
			if decompressor == nil {
				decompressor, err = gzip.NewReader(buffer)
				defer decompressor.Close()
				if err != nil {
					return err
				}
				zipReader = bufio.NewReader(decompressor)
			}
			data = make([]byte, 512, 512)
			_, err = zipReader.Read(data)
			if err != nil {
				return err
			}
			strBuffer := bytes.NewBuffer(data)
			io.CopyN(writer, strBuffer, int64(len(data)))
		}
		if c.conf.TTL > 0 {
			if time.Now().Sub(start).Nanoseconds() > c.conf.TTL {
				return nil
			}
		}
	}
	return err
}

// Initializes a TLS net.Conn object to the configured server.
func (c *Connection) connect() error {
	var addr string
	var conn net.Conn
	var err error
	if c.conf.Proxy == "" {
		addr = fmt.Sprintf("%v:443", c.conf.URL.Host)
		conn, err = tls.Dial("tcp", addr, nil)
	} else {
		addr = c.conf.Proxy
		conn, err = net.Dial("tcp", addr)
	}
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

// Sends a signed HTTP request along an opened connection.
func (c *Connection) request() error {
	if c.writer == nil {
		return errors.New("Writer is not initialized")
	}
	reqUrl := fmt.Sprintf("%v://%v%v", c.conf.URL.Scheme, c.conf.URL.Host, c.conf.URL.Path)
	req, err := http.NewRequest(c.conf.Method, reqUrl, nil)
	if err != nil {
		return err
	}
	if !c.conf.Chunked {
		// Send Connection: close, which mimics HTTP 1.0 behavior.
		req.Header.Set("Connection", "close")
	}
	if c.conf.GZip {
		req.Header.Set("Accept-Encoding", "deflate, gzip")
	}
	user := oauth1a.NewAuthorizedConfig(c.cred.Token, c.cred.Secret)
	service := &oauth1a.Service{
		ClientConfig: &oauth1a.ClientConfig{
			ConsumerKey:    c.cred.ConsumerKey,
			ConsumerSecret: c.cred.ConsumerSecret,
		},
		Signer: new(oauth1a.HmacSha1Signer),
	}
	if err := service.Sign(req, user); err != nil {
		return err
	}
	if c.conf.Proxy == "" {
		err = req.Write(c.writer)
	} else {
		err = req.WriteProxy(c.writer)
	}
	return err
}
