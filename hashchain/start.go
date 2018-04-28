package hashchain

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"

	"github.com/frankbraun/codechain/internal/base64"
	"github.com/frankbraun/codechain/util/file"
	"github.com/frankbraun/codechain/util/lockfile"
	"github.com/frankbraun/codechain/util/time"
	"golang.org/x/crypto/ed25519"
)

// Start returns a new hash chain with signature control list m.
func Start(filename string, secKey [64]byte, comment []byte) (*HashChain, string, error) {
	// check arguments
	exists, err := file.Exists(filename)
	if err != nil {
		return nil, "", err
	}
	if exists {
		return nil, "", fmt.Errorf("hashchain: file '%s' exists already", filename)
	}

	// init
	var c HashChain
	c.lock, err = lockfile.Create(filename)
	if err != nil {
		return nil, "", err
	}
	c.fp, err = os.Create(filename)
	if err != nil {
		return nil, "", err
	}

	// create signature
	var nonce [24]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return nil, "", err
	}
	pub := secKey[32:]
	msg := append(pub, nonce[:]...)
	if len(comment) > 0 {
		msg = append(msg, comment...)
	}
	sig := ed25519.Sign(secKey[:], msg)

	// create entry
	typeFields := []string{
		base64.Encode(pub),
		base64.Encode(nonce[:]),
		base64.Encode(sig[:]),
	}
	if len(comment) > 0 {
		typeFields = append(typeFields, string(comment))
	}
	l := &link{
		previous:   emptyTree,
		datum:      time.Now(),
		linkType:   chainStartType,
		typeFields: typeFields,
	}
	c.chain = append(c.chain, l)

	// verify
	if err := c.verify(); err != nil {
		return nil, "", err
	}

	// save
	entry := l.String()
	if _, err := fmt.Fprintln(c.fp, entry); err != nil {
		return nil, "", err
	}
	return &c, entry, nil
}