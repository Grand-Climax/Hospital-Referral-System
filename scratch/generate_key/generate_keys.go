package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

func main() {
	k1 := make([]byte, 32)
	rand.Read(k1)
	fmt.Println("KEY=" + base64.StdEncoding.EncodeToString(k1))
}
