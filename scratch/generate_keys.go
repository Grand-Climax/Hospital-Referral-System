package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

func main() {
	k1 := make([]byte, 32)
	rand.Read(k1)
	k2 := make([]byte, 32)
	rand.Read(k2)
	fmt.Println("PATIENT_AES_KEY=" + base64.StdEncoding.EncodeToString(k1))
	fmt.Println("PATIENT_HMAC_KEY=" + base64.StdEncoding.EncodeToString(k2))
}
