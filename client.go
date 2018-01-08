/*
Implements the solution to assignment 1 for UBC CS 416 2017 W2.

Usage:
$ go run client.go [local UDP ip:port] [local TCP ip:port] [aserver UDP ip:port]

Example:
$ go run client.go 127.0.0.1:2020 127.0.0.1:3030 127.0.0.1:7070

*/

package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"strconv"
	//"encoding/json"

	// TODO
	"fmt"
	"math"
	"net"
	"os"
)

/////////// Msgs used by both auth and fortune servers:

// An error message from the server.
type ErrMessage struct {
	Error string
}

/////////// Auth server msgs:

// Message containing a nonce from auth-server.
type NonceMessage struct {
	Nonce string
	N     int64 // PoW difficulty: number of zeroes expected at end of md5(nonce+secret)
}

// Message containing an the secret value from client to auth-server.
type SecretMessage struct {
	Secret string
}

// Message with details for contacting the fortune-server.
type FortuneInfoMessage struct {
	FortuneServer string // TCP ip:port for contacting the fserver
	FortuneNonce  int64
}

/////////// Fortune server msgs:

// Message requesting a fortune from the fortune-server.
type FortuneReqMessage struct {
	FortuneNonce int64
}

// Response from the fortune-server containing the fortune.
type FortuneMessage struct {
	Fortune string
	Rank    int64 // Rank of this client solution
}

// Main workhorse method.
func main() {
	// TODO
	args := os.Args[1:]
	if len(args) != 3 {
		fmt.Println("Usage: client.go [local UDP ip:port] [local TCP ip:port] [aserver UDP ip:port]")
		return
	}
	localUDPPort := args[0]
	// localTCPPort := args[1]
	remoteAserverPort := args[2]
	buffer := make([]byte, 1024)

	conn := getUDPConnection(localUDPPort)
	aserver, err := net.ResolveUDPAddr("udp", remoteAserverPort)
	if err != nil {
		fmt.Println(err)
	}
	payload, _ := json.Marshal("hi, I want the goods")
	conn.WriteToUDP(payload, aserver)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println(err)
	}
	res := buffer[0:n]
	var dat map[string]interface{}
	json.Unmarshal(res, &dat)
	var nFloat = dat["N"].(float64)
	var nInt = int64(nFloat)
	nonceMsg := NonceMessage{
		Nonce: dat["Nonce"].(string),
		N:     nInt,
	}
	fmt.Println(nonceMsg)
	sMsg := SecretMessage{
		Secret: "",
	}
	for i := 0; i < math.MaxInt32; i++ {
		str := computeNonceSecretHash(nonceMsg.Nonce, strconv.Itoa(i))
		powisDone := false
		for suffix := len(str) - int(nonceMsg.N) - 1; suffix < len(str); suffix++ {
			if str[suffix] == 48 {
				// fmt.Println(str[suffix])
				powisDone = true
			} else {
				powisDone = false
				break
			}
		}
		if powisDone {
			fmt.Println(str)
			sMsg.Secret = strconv.Itoa(i)
			break
		}
	}
	fmt.Println(sMsg)
	payloadSecret, _ := json.Marshal(sMsg)
	conn.WriteToUDP(payloadSecret, aserver)
	bufferSecret := make([]byte, 1024)
	n2, err := conn.Read(bufferSecret)
	if err != nil {
		fmt.Println(err)
	}
	resSecret := bufferSecret[0:n2]
	var datSecret map[string]interface{}
	json.Unmarshal(resSecret, &datSecret)
	fmt.Println(datSecret)
	var nFortFloat = datSecret["FortuneNonce"].(float64)
	var nFortInt = int64(nFortFloat)
	infoMsg := FortuneInfoMessage{
		FortuneServer: datSecret["FortuneServer"].(string),
		FortuneNonce:  nFortInt,
	}
	// listenConn, _ := net.Listen("tcp", localTCPPort)
	fserver, err := net.Dial("tcp", infoMsg.FortuneServer)
	if err != nil {
		fmt.Println(err)
	}
	fortReq := FortuneReqMessage{
		FortuneNonce: infoMsg.FortuneNonce,
	}
	payloadFort, _ := json.Marshal(fortReq)
	fserver.Write(payloadFort)
	bufferFort := make([]byte, 1024)
	n3, err := fserver.Read(bufferFort)
	if err != nil {
		fmt.Println(err)
	}
	resFort := bufferFort[0:n3]
	var datFort map[string]interface{}
	json.Unmarshal(resFort, &datFort)
	fmt.Println(datFort)
	// Use json.Marshal json.Unmarshal for encoding/decoding to servers

}

func getUDPConnection(ip string) *net.UDPConn {
	addr, err := net.ResolveUDPAddr("udp", ip)
	if err != nil {
		fmt.Println(err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println(err)
	}
	return conn
}

// Returns the MD5 hash as a hex string for the (nonce + secret) value.
func computeNonceSecretHash(nonce string, secret string) string {
	h := md5.New()
	h.Write([]byte(nonce + secret))
	str := hex.EncodeToString(h.Sum(nil))
	return str
}
