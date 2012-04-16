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

package twurlrc

import (
	"testing"
)

var SAMPLE string = `
--- 
configuration: 
  default_profile: 
  - userA
  - CONSUMERKEYA
profiles: 
  userA: 
    CONSUMERKEYA: 
      consumer_secret: CONSUMERSECRETA
      username: userA
      consumer_key: CONSUMERKEYA
      secret: SECRETA
      token: TOKENA
  userB:
    CONSUMERKEYB:
      consumer_secret: CONSUMERSECRETB
      username: userB
      consumer_key: CONSUMERKEYB
      secret: SECRETB
      token: TOKENB
    CONSUMERKEYC:
      consumer_secret: CONSUMERSECRETC
      username: userB
      consumer_key: CONSUMERKEYC
      secret: SECRETC
      token: TOKENC`

// Compares two arrays for set equality (both have same members).
func Compare(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	set := map[string]struct{}{}
	for _, val := range a {
		set[val] = struct{}{}
	}
	for _, val := range b {
		if _, ok := set[val]; ok == false {
			return false
		}
	}
	return true
}

// Tests parting a piece of sample data.
func TestParse(t *testing.T) {
	twrc, err := Parse(SAMPLE)
	if err != nil {
		t.Error(err.Error())
	}
	profiles := twrc.GetProfiles()
	if !Compare(profiles, []string{"userA", "userB"}) {
		t.Error("GetProfiles did not return correct data")
		t.Log(profiles)
	}
	keys := twrc.GetKeys("userB")
	if !Compare(keys, []string{"CONSUMERKEYB", "CONSUMERKEYC"}) {
		t.Error("GetKeys did not return correct data")
		t.Log(keys)
	}
	cred := twrc.GetDefaultCredentials()
	if cred.Token != "TOKENA" {
		t.Error("Invalid default token")
	}
	if cred.Username != "userA" {
		t.Error("Invalid default user")
	}
	if cred.ConsumerKey != "CONSUMERKEYA" {
		t.Error("Invalid default consumer key")
	}
	if cred.ConsumerSecret != "CONSUMERSECRETA" {
		t.Error("Invalid default consumer secret")
	}
	if cred.Secret != "SECRETA" {
		t.Error("Invalid default secret")
	}
	cred = twrc.GetCredentials("userB", "CONSUMERKEYC")
	if cred.Token != "TOKENC" {
		t.Error("Invalid requested token")
	}
	if cred.Username != "userB" {
		t.Error("Invalid requested user")
	}
	if cred.ConsumerKey != "CONSUMERKEYC" {
		t.Error("Invalid requested consumer key")
	}
	if cred.ConsumerSecret != "CONSUMERSECRETC" {
		t.Error("Invalid requested consumer secret")
	}
	if cred.Secret != "SECRETC" {
		t.Error("Invalid requested secret")
	}
}
