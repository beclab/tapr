package uid

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"k8s.io/klog/v2"
	"time"
)

// uid returns a unique id. These ids consist of 128 bits from a
// cryptographically strong pseudo-random generator and are like uuids, but
// without the dashes and significant bits.
//
// See: http://en.wikipedia.org/wiki/UUID#Random_UUID_probability_of_duplicates
func Uid() string {
	id := make([]byte, 16)
	_, err := io.ReadFull(rand.Reader, id)
	if err != nil {
		// This is probably an appropriate way to handle errors from our source
		// for random bits.
		panic(err)
	}
	return hex.EncodeToString(id)
}

func MakeUid(filePath string) string {
	hash := md5.Sum([]byte(filePath))
	md5String := hex.EncodeToString(hash[:])
	klog.Infof("filePath:%s, uid:%s", filePath, md5String)
	return md5String
}

func GenerateUniqueIdentifier(relativePath string) string {
	h := md5.New()
	io.WriteString(h, relativePath+time.Now().String())
	return fmt.Sprintf("%x%s", h.Sum(nil), relativePath)
}
