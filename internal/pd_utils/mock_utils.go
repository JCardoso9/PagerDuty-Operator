package pd_utils

import "math/rand"

const letterBytes = "abcdefghijklmnopqrstuvwxyz"

func RandStr(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}
