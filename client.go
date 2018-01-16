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
	"math/rand"
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
	udpAddr, _ := net.ResolveUDPAddr("udp", localUDPPort)
	conn, _ := net.ListenUDP("udp", udpAddr)

	aserver, _ := net.ResolveUDPAddr("udp", remoteAserverPort)
	payload, _ := json.Marshal("hi, I want the goods")
	conn.WriteToUDP(payload, aserver)

	// UDP Read Nonce Message
	n, _ := conn.Read(buffer)

	res := buffer[0:n]
	var nonceMsg NonceMessage
	json.Unmarshal(res, &nonceMsg)

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

	quitChan := make(chan bool)
	go getSecretRand(nonceMsg, dataChan, arrayNonce, quitChan)
	go getSecret64(nonceMsg, dataChan, arrayNonce, quitChan)
	// Bottom up
	for i := 0; i < (math.MaxInt32-1)/2; i = i + 32537631 {
		go getSecretRange(i, i+32537631, nonceMsg, dataChan, arrayNonce)
	}
	// Top down
	for i := (math.MaxInt32 - 1); i > (math.MaxInt32-1)/2; i = i - 32537631 {
		go getSecretRange(i, i-32537631, nonceMsg, dataChan, arrayNonce)
	}
	// middle up
	for i := (math.MaxInt32 - 1) / 2; i < (math.MaxInt32 - 1); i = i + 32537631 {
		go getSecretRange(i, i-32537631, nonceMsg, dataChan, arrayNonce)
	}
	// middle down
	for i := (math.MaxInt32 - 1) / 2; i > 0; i = i - 32537631 {
		go getSecretRange(i, i-32537631, nonceMsg, dataChan, arrayNonce)
	}
	for i := 0; i < (math.MaxInt32-1)/2; i = i + 32537631 {
		go getSecretRange(i, i+32537631, nonceMsg, dataChan, arrayNonce)
	}

	sMsg.Secret = <-dataChan
	close(quitChan)

	// UDP Send Secret
	payloadSecret, _ := json.Marshal(sMsg)
	conn.WriteToUDP(payloadSecret, aserver)
	bufferSecret := make([]byte, 1024)

	// UDP Read FServer Info
	n2, _ := conn.Read(bufferSecret)

	resSecret := bufferSecret[0:n2]
	var infoMessage FortuneInfoMessage
	json.Unmarshal(resSecret, &infoMessage)

	// TCP Send Fortune Req Message
	localAddr, _ := net.ResolveTCPAddr("tcp", localTCPPort)

	fserver, _ := net.ResolveTCPAddr("tcp", infoMessage.FortuneServer)

	fConn, _ := net.DialTCP("tcp", localAddr, fserver)

	fortReq := FortuneReqMessage{
		FortuneNonce: infoMessage.FortuneNonce,
	}
	payloadFort, _ := json.Marshal(fortReq)
	fConn.Write(payloadFort)

	// TCP Read Fortune Message
	bufferFort := make([]byte, 1024)
	n3, _ := fConn.Read(bufferFort)

	resFort := bufferFort[0:n3]
	var fMsg FortuneMessage
	json.Unmarshal(resFort, &fMsg)
	fmt.Println(fMsg.Fortune)

	conn.Close()
	fConn.Close()
}

func getSecretRange(start, end int, nonceMsg NonceMessage, dataChan chan<- string, arrayNonce string) {
	for i := start; i < end; i++ {
		str := computeNonceSecretHash(nonceMsg.Nonce, strconv.Itoa(i))
		suffix := len(str) - int(nonceMsg.N)
		if str[suffix:] == arrayNonce {
			dataChan <- strconv.Itoa(i)
			return
		}
	}
}
func getSecretRand(nonceMsg NonceMessage, dataChan chan<- string, arrayNonce string, quitChan <-chan bool) {
	for {
		select {
		case _ = <-quitChan:
			return
		default:
			var randI = strconv.FormatInt(rand.Int63(), 36)
			str := computeNonceSecretHash(nonceMsg.Nonce, randI)
			suffix := len(str) - int(nonceMsg.N)
			if str[suffix:] == arrayNonce {
				dataChan <- randI
				return
			}
		}
	}
}
func getSecret64(nonceMsg NonceMessage, dataChan chan<- string, arrayNonce string, quitChan <-chan bool) {
	for i := (math.MaxInt64 - 1); i > math.MaxInt32-1; i = i - 32537631 {
		select {
		case _ = <-quitChan:
			return
		default:
			var randI = strconv.FormatInt(rand.Int63(), 36)
			str := computeNonceSecretHash(nonceMsg.Nonce, randI)
			suffix := len(str) - int(nonceMsg.N)
			if str[suffix:] == arrayNonce {
				dataChan <- randI
				return
			}
		}
	}
}

// Returns the MD5 hash as a hex string for the (nonce + secret) value.
func computeNonceSecretHash(nonce string, secret string) string {
	h := md5.New()
	h.Write([]byte(nonce + secret))
	str := hex.EncodeToString(h.Sum(nil))
	return str
}
