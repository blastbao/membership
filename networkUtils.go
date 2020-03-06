package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

// Given a Listener and a channel, this function will receive data from the socket,
// decodes it to Message struct and send it through the channel. Used as a goroutine
// in Leader and follower event loops.
func acceptTCPMessages(tcp net.Listener, inMsgChan chan InMsgType) {

	buf := make([]byte, 1024)

	for {
		// 接收连接
		conn, err := tcp.Accept()
		LogFatalCheck(err, "Cannot accept connection.")

		// 读取消息
		_, err = conn.Read(buf)
		LogFatalCheck(err, "Error reading from tcp socket")

		// 消息解码
		msg := new(Message)
		gobj := gob.NewDecoder(bytes.NewBuffer(buf))
		gobj.Decode(msg)

		// 获取对方ip
		remoteIP := strings.Split(conn.RemoteAddr().String(), ":")[0]

		// 消息写入管道
		inMsgChan <- InMsgType{From: remoteIP, Message: *msg}

		// conn 退出作用域，被析构，相当于当前连接是短连接
	}

}

// Send Message to a process using TCP socket
func sendTCPMsg(m Message, toAddr string) {


	// 建立 tcp 连接
	conn, err := net.Dial("tcp", toAddr)
	if err != nil {
		log.Printf("Error establishing tcp connection to %s: Reason: %v", toAddr, err)
		return
	}
	defer conn.Close()

	// 发送 msg 消息
	buf := new(bytes.Buffer)
	gobobj := gob.NewEncoder(buf)
	gobobj.Encode(m)
	conn.Write(buf.Bytes())
}

// Listen for heartbeats and send the hostname through channel
func monitorUDPHeartbeats(udp net.UDPConn, hbMsgChan chan string) {

	for {
		// 读取 udp 消息，该消息代表心跳，不用关心内容，直接写入管道
		buf := make([]byte, 8)
		_, remoteAddr, _ := udp.ReadFromUDP(buf)
		hbMsgChan <- remoteAddr.IP.String()
	}
}

func sendHeartbeat(toAddr string) {
	// 建立 udp 连接
	conn, err := net.Dial("udp", toAddr)
	LogFatalCheck(err, fmt.Sprintf("Error establishing udp connection to %s", toAddr))
	defer conn.Close()
	// 发送心跳消息
	conn.Write([]byte("ping"))
}

// 广播心跳到其它节点
func multicastHeartbeats() {
	// 遍历其它节点，逐个发送心跳
	for h := range membershipList {
		addr := fmt.Sprintf("%s:%d", h, port+1)
		go sendHeartbeat(addr)
	}
}

// 每隔 xxx 秒给其它节点发送心跳
func startHeartbeat(frequency int) {
	for {
		multicastHeartbeats()
		time.Sleep(time.Duration(frequency) * time.Second)
	}
}

// 定时发送信号，触发 gc
func startHBGCTimer(timerMsgChan chan bool) {
	for {
		time.Sleep(time.Duration(3) * time.Second)
		timerMsgChan <- true
	}
}

// Send message to all hosts in membership list.
func multicastTCPMessage(msg Message, exceptHost string) {

	//广播消息给除了 exceptHost 之外的每个节点
	for h := range membershipList {

		// ignore failed node
		if h == exceptHost {
			continue
		}

		addr := fmt.Sprintf("%s:%d", h, port)
		go sendTCPMsg(msg, addr)
	}
}
