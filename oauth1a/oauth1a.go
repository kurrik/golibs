// Copyright 2011 Arne Roomann-Kurrik.
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

/*
	Package oauth1a implements the OAuth 1.0a specification.
*/
package oauth1a

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Container for client-specific configuration related to the OAuth process.
// This struct is intended to be serialized and stored for future use.
type ClientConfig struct {
	ConsumerSecret string
	ConsumerKey    string
	CallbackURL    string
}

// Represents an API which offers OAuth access.
type Service struct {
	RequestURL   string
	AuthorizeURL string
	AccessURL    string
	*ClientConfig
	Signer
}

// Signs an HTTP request with the needed OAuth parameters.
func (s *Service) Sign(request *http.Request, userConfig *UserConfig) error {
	return s.Signer.Sign(request, s.ClientConfig, userConfig)
}

// Interface for any OAuth signing implementations.
type Signer interface {
	Sign(request *http.Request, config *ClientConfig, user *UserConfig) error
}

// A Signer which implements the HMAC-SHA1 signing algorithm.
type HmacSha1Signer struct{}

// Sort a set of request parameters alphabetically, and encode according to the
// OAuth 1.0a specification.
func (HmacSha1Signer) encodeParameters(params map[string]string) string {
	keys := make([]string, len(params))
	encodedParts := make([]string, len(params))
	i := 0
	for key, _ := range params {
		keys[i] = key
		i += 1
	}
	sort.Strings(keys)
	for i, key := range keys {
		value := params[key]
		encoded := Rfc3986Escape(key) + "=" + Rfc3986Escape(value)
		encodedParts[i] = encoded
	}
	return url.QueryEscape(strings.Join(encodedParts, "&"))
}

// Generate a unique nonce value.  Should not be called more than once per
// nanosecond
// TODO: Come up with a better generation method.
func (HmacSha1Signer) GenerateNonce() string {
	ns := time.Now()
	token := fmt.Sprintf("OAuth Client Lib %v", ns)
	h := sha1.New()
	h.Write([]byte(token))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// Returns a map of all of the oauth_* (including signature) parameters for the
// given request, and the signature base string used to generate the signature.
func (s *HmacSha1Signer) GetOAuthParams(request *http.Request, clientConfig *ClientConfig, userConfig *UserConfig, nonce string, timestamp string) (map[string]string, string) {
	request.ParseForm()
	oauthParams := map[string]string{
		"oauth_consumer_key":     clientConfig.ConsumerKey,
		"oauth_nonce":            nonce,
		"oauth_signature_method": "HMAC-SHA1",
		"oauth_timestamp":        timestamp,
		"oauth_version":          "1.0",
	}
	tokenKey, tokenSecret := userConfig.GetToken()
	if tokenKey != "" {
		oauthParams["oauth_token"] = tokenKey
	}
	signingParams := map[string]string{}
	for key, value := range oauthParams {
		signingParams[key] = value
	}
	for key, value := range request.URL.Query() {
		//TODO: Support multiple parameters with the same name.
		signingParams[key] = value[0]
	}
	for key, value := range request.Form {
		//TODO: Support multiple parameters with the same name.
		signingParams[key] = value[0]
	}
	signingUrl := fmt.Sprintf("%v://%v%v", request.URL.Scheme, request.URL.Host, request.URL.Path)
	signatureParts := []string{
		request.Method,
		url.QueryEscape(signingUrl),
		s.encodeParameters(signingParams)}
	signatureBase := strings.Join(signatureParts, "&")
	oauthParams["oauth_signature"] = s.GetSignature(clientConfig.ConsumerSecret, tokenSecret, signatureBase)
	return oauthParams, signatureBase
}

// Calculates the HMAC-SHA1 signature of a base string, given a consumer and
// token secret.
func (s *HmacSha1Signer) GetSignature(consumerSecret string, tokenSecret string, signatureBase string) string {
	signingKey := consumerSecret + "&" + tokenSecret
	signer := hmac.New(sha1.New, []byte(signingKey))
	signer.Write([]byte(signatureBase))
	oauthSignature := base64.StdEncoding.EncodeToString(signer.Sum(nil))
	return oauthSignature
}

// Given an unsigned request, add the appropriate OAuth Authorization header
// using the HMAC-SHA1 algorithm.
func (s *HmacSha1Signer) Sign(request *http.Request, clientConfig *ClientConfig, userConfig *UserConfig) error {
	nonce := s.GenerateNonce()
	timestamp := fmt.Sprintf("%v", time.Now())
	oauthParams, _ := s.GetOAuthParams(request, clientConfig, userConfig, nonce, timestamp)
	headerParts := make([]string, len(oauthParams))
	var i = 0
	for key, value := range oauthParams {
		headerParts[i] = Rfc3986Escape(key) + "=\"" + Rfc3986Escape(value) + "\""
		i += 1
	}
	oauthHeader := "OAuth " + strings.Join(headerParts, ", ")
	request.Header["Authorization"] = []string{oauthHeader}
	return nil
}

// Characters which should not be escaped according to RFC 3986.
const UNESCAPE_CHARS = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-._~"

// Escapes a string more in line with Rfc3986 than http.URLEscape.
// URLEscape was converting spaces to "+" instead of "%20", which was messing up
// the signing of requests.
func Rfc3986Escape(input string) string {
	var output bytes.Buffer
	// Convert string to bytes because iterating over a unicode string
	// in go parses runes, not bytes.
	for _, c := range []byte(input) {
		if strings.IndexAny(string(c), UNESCAPE_CHARS) == -1 {
			encoded := fmt.Sprintf("%%%X", c)
			output.Write([]uint8(encoded))
		} else {
			output.WriteByte(uint8(c))
		}
	}
	return string(output.Bytes())
}
