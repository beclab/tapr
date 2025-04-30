package utils

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"os"
)

func MD5(s string) string {
	hash := md5.Sum([]byte(s))
	return hex.EncodeToString(hash[:])
}

func Hex(data []byte) string {
	buf := make([]byte, hex.EncodedLen(len(data)))
	hex.Encode(buf, data)
	return string(buf)
}

func ValueMust[R any](v R, err error) R {
	if err != nil {
		panic(err)
	}

	return v
}

func ListContains[T comparable](items []T, v T) bool {
	for _, item := range items {
		if v == item {
			return true
		}
	}
	return false
}

func AggregateErrs(errs []error) error {
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
		var errStr string
		for _, e := range errs {
			errStr += e.Error() + "\t"
		}
		return errors.New(errStr[:len(errStr)-1])
	}
}

func CheckDirExist(dirPath string) bool {
	fi, err := os.Stat(dirPath)
	return (err == nil || os.IsExist(err)) && fi.IsDir()
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}

	if os.IsNotExist(err) {
		return false
	}
	return false
}

func PathExistsAndGetLen(path string) (bool, int64) {
	info, err := os.Stat(path)
	if err == nil {
		return true, info.Size()
	}

	if os.IsNotExist(err) {
		return false, 0
	}
	return false, 0
}
