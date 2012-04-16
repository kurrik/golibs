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
	"io/ioutil"
	"launchpad.net/goyaml"
	"os"
)

// Represents OAuth credentials to make requests on behalf of a user.
type Credentials struct {
	Token          string
	Username       string
	ConsumerKey    string
	ConsumerSecret string
	Secret         string
}

// Returns a path to the default twurlrc location.
func GetDefaultPath() string {
	return os.ExpandEnv("$HOME/.twurlrc")
}

// Represents a parsed ~/.twurlrc formatted file.
type Twurlrc struct {
	data map[string]interface{}
}

// Given the contents of a .twurlrc file, return a parsed data structure.
func Parse(text string) (*Twurlrc, error) {
	t := new(Twurlrc)
	t.data = make(map[string]interface{})
	err := goyaml.Unmarshal([]uint8(text), t.data)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// Given a path to a twurlrc file, return a parsed data structure.
func Load(path string) (*Twurlrc, error) {
	text, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(string(text))
}

// Returns credentials for the given user profile and consumer key.
func (t *Twurlrc) GetCredentials(profile string, key string) *Credentials {
	profileMap := t.data["profiles"].(map[interface{}]interface{})
	keyMap := profileMap[profile].(map[interface{}]interface{})
	data := keyMap[key].(map[interface{}]interface{})
	return &Credentials{
		Token:          data["token"].(string),
		Username:       data["username"].(string),
		ConsumerKey:    data["consumer_key"].(string),
		ConsumerSecret: data["consumer_secret"].(string),
		Secret:         data["secret"].(string),
	}
}

// Returns the default credentials, as specified in the ~/.twurlrc file.
func (t *Twurlrc) GetDefaultCredentials() *Credentials {
	configMap := t.data["configuration"].(map[interface{}]interface{})
	parts := configMap["default_profile"].([]interface{})
	return t.GetCredentials(parts[0].(string), parts[1].(string))
}

// Returns a list of consumer keys authorized with the given profile.
func (t *Twurlrc) GetKeys(profile string) []string {
	profileMap := t.data["profiles"].(map[interface{}]interface{})
	keyMap := profileMap[profile].(map[interface{}]interface{})
	keys := make([]string, len(keyMap))
	i := 0
	for key, _ := range keyMap {
		keys[i] = key.(string)
		i++
	}
	return keys
}

// Returns a list of profiles listed in the ~/.twurlrc file.
func (t *Twurlrc) GetProfiles() []string {
	profileMap := t.data["profiles"].(map[interface{}]interface{})
	profiles := make([]string, len(profileMap))
	i := 0
	for key, _ := range profileMap {
		profiles[i] = key.(string)
		i++
	}
	return profiles
}
