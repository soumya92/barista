// Copyright 2018 Google Inc.
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

package oauth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"os"
	"sync"

	"github.com/spf13/afero"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/oauth2"
)

var (
	globalEncryptionKey   []byte
	globalEncryptionKeyMu sync.Mutex
)

const (
	// Number of iterations to use when deriving AES keys.
	pbkdf2Iterations = 4096
	// Size of IV for AES-256 keys.
	aes256KeySize = 32
)

// SetEncryptionKey sets the global encryption key for the oauth package.
// All site-specific encryption keys are derived from the global encryption key.
// For that reason this is a very high-value key, and should be stored properly,
// for example, using libsecret.
func SetEncryptionKey(key []byte) {
	globalEncryptionKeyMu.Lock()
	defer globalEncryptionKeyMu.Unlock()
	if len(globalEncryptionKey) != 0 {
		panic("Encryption key already set")
	}
	globalEncryptionKey = key
}

func getEncryptionKeyChecked() []byte {
	globalEncryptionKeyMu.Lock()
	defer globalEncryptionKeyMu.Unlock()
	if len(globalEncryptionKey) == 0 {
		panic("Encryption key not set")
	}
	return globalEncryptionKey
}

var fs = afero.NewOsFs()

type encryptedToken struct{ Salt, IV, Token []byte }

func loadToken(filename string) (*oauth2.Token, error) {
	key := getEncryptionKeyChecked()
	f, err := fs.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	eTok := encryptedToken{}
	err = json.NewDecoder(f).Decode(&eTok)
	if err != nil {
		return nil, err
	}
	dk := pbkdf2.Key(key, eTok.Salt, pbkdf2Iterations, aes256KeySize, sha256.New)
	block, _ := aes.NewCipher(dk) // no error, key size is fixed.
	cipher.NewCFBDecrypter(block, eTok.IV).XORKeyStream(eTok.Token, eTok.Token)
	tok := &oauth2.Token{}
	err = json.Unmarshal(eTok.Token, tok)
	return tok, err
}

var randRead = rand.Read // for tests.

func storeToken(filename string, token *oauth2.Token) error {
	key := getEncryptionKeyChecked()
	eTok := encryptedToken{
		Salt: make([]byte, 64),
		IV:   make([]byte, aes.BlockSize),
	}
	var err error
	eTok.Token, _ = json.Marshal(token) // no error, input is only []bytes.
	if _, err := randRead(eTok.Salt); err != nil {
		return err
	}
	ek := pbkdf2.Key(key, eTok.Salt, pbkdf2Iterations, aes256KeySize, sha256.New)
	block, _ := aes.NewCipher(ek) // no error, key size is fixed.
	if _, err := randRead(eTok.IV); err != nil {
		return err
	}
	cipher.NewCFBEncrypter(block, eTok.IV).XORKeyStream(eTok.Token, eTok.Token)
	f, err := fs.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err == nil {
		defer f.Close()
		err = json.NewEncoder(f).Encode(eTok)
	}
	return err
}
