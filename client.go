/*
Implements the solution to assignment 1 for UBC CS 416 2017 W2.

Usage:
$ go run client.go [local UDP ip:port] [local TCP ip:port] [aserver UDP ip:port]

Example:
$ go run client.go 127.0.0.1:2020 127.0.0.1:3030 127.0.0.1:7070

*/

package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"math"
	"strconv"
	//"encoding/json"

	// TODO
	"fmt"
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
	args := os.Args[1:]
	if len(args) != 3 {
		return
	}
	localUDPPort := args[0]
	localTCPPort := args[1]
	remoteAserverPort := args[2]
	buffer := make([]byte, 1024)

	// UDP Send Arbitrary Message
	conn := getUDPConnection(localUDPPort)
	aserver, err := net.ResolveUDPAddr("udp", remoteAserverPort)
	if err != nil {
		//fmt.Println(err)
	}
	payload, _ := json.Marshal("hi, I want the goods")
	conn.WriteToUDP(payload, aserver)

	// UDP Read Nonce Message
	n, err := conn.Read(buffer)
	if err != nil {
		//fmt.Println(err)
	}
	res := buffer[0:n]
	var nonceMsg NonceMessage
	json.Unmarshal(res, &nonceMsg)
	fmt.Println(nonceMsg)
	// Compute Secret
	sMsg := SecretMessage{
		Secret: "",
	}
	var bufferString bytes.Buffer
	for i := 0; i < int(nonceMsg.N); i++ {
		bufferString.WriteString("0")
	}
	arrayNonce := bufferString.String()
	dataChan := make(chan string)

	for i := 0; i < (math.MaxInt32-1)/2; i = i + 1015839 {
		go getSecret(i, i+1015839, nonceMsg, dataChan, arrayNonce)
		fmt.Print("Bottom  ", i)
		fmt.Println(" to ", i+1015839)
	}
	for i := (math.MaxInt32 - 1); i > (math.MaxInt32-1)/2; i = i - 1015839 {
		go getSecret(i, i-1015839, nonceMsg, dataChan, arrayNonce)
		fmt.Print("Top  ", i)
		fmt.Println(" to ", i-1015839)
	}

	// go getSecret(0, math.MaxInt32/4, nonceMsg, dataChan, arrayNonce)
	// go getSecret(math.MaxInt32/4+1, math.MaxInt32/2, nonceMsg, dataChan, arrayNonce)
	// go getSecret(math.MaxInt32/2+1, (math.MaxInt32/4)*3, nonceMsg, dataChan, arrayNonce)
	// go getSecret((math.MaxInt32/4)*3+1, math.MaxInt32, nonceMsg, dataChan, arrayNonce)

	sMsg.Secret = <-dataChan

	// UDP Send Secret
	payloadSecret, _ := json.Marshal(sMsg)
	conn.WriteToUDP(payloadSecret, aserver)
	bufferSecret := make([]byte, 1024)

	// UDP Read FServer Info
	n2, err := conn.Read(bufferSecret)
	if err != nil {
		// fmt.Println(err)
	}
	resSecret := bufferSecret[0:n2]
	var infoMessage FortuneInfoMessage
	json.Unmarshal(resSecret, &infoMessage)

	// TCP Send Fortune Req Message
	localAddr, err := net.ResolveTCPAddr("tcp", localTCPPort)
	if err != nil {
		// fmt.Println(err)
	}
	fserver, err := net.ResolveTCPAddr("tcp", infoMessage.FortuneServer)
	if err != nil {
		// fmt.Println(err)
	}
	fConn, err := net.DialTCP("tcp", localAddr, fserver)
	if err != nil {
		// fmt.Println(err)
	}
	fortReq := FortuneReqMessage{
		FortuneNonce: infoMessage.FortuneNonce,
	}
	payloadFort, _ := json.Marshal(fortReq)
	fConn.Write(payloadFort)

	// TCP Read Fortune Message
	bufferFort := make([]byte, 1024)
	n3, err := fConn.Read(bufferFort)
	if err != nil {
		// fmt.Println(err)
	}
	resFort := bufferFort[0:n3]
	var fMsg FortuneMessage
	json.Unmarshal(resFort, &fMsg)
	fmt.Println(fMsg.Fortune)

	conn.Close()
	fConn.Close()
}

func getSecret(start, end int, nonceMsg NonceMessage, dataChan chan<- string, arrayNonce string) {
	for i := start; i < end; i++ {
		str := computeNonceSecretHash(nonceMsg.Nonce, strconv.Itoa(i))
		suffix := len(str) - int(nonceMsg.N)
		if str[suffix:] == arrayNonce {
			dataChan <- strconv.Itoa(i)
			return
		}
	}
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
