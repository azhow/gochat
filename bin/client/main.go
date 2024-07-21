package main

import (
	"bufio"
	"fmt"
	"gocc/gochat/pkg/utils"
	"io"
	"net"
	"os"
)

func main() {
	connection, err := net.Dial("tcp", utils.IpAddr+":"+utils.SrvPort)
	if err != nil {
		fmt.Println("Error attempting to connect to server: ", err)
		return
	}

	handle(connection)
}

func handle(conn net.Conn) {
	defer conn.Close()

	stdinChan := make(chan string)
	srvChan := make(chan string)

	go readIntoChannel(os.Stdin, stdinChan)
	go readFromServerIntoChannel(conn, srvChan)

	quit := false
	for !quit {
		select {
		case input := <-stdinChan:
			if len(input) == 0 {
				fmt.Println("client quit")
				quit = true
			} else {
				conn.Write([]byte(input))
			}
		case srvMessage := <-srvChan:
			if len(srvMessage) == 0 {
				fmt.Println("server quit")
				quit = true
			} else {
				fmt.Println(srvMessage)
			}

		}
	}
}

func readIntoChannel(r io.Reader, c chan string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		c <- scanner.Text()
	}

	close(c)
}

func readFromServerIntoChannel(conn net.Conn, c chan string) {
	quit := false
	for !quit {
		buff := make([]byte, 1024)

		numBytesReceived, err := conn.Read(buff[0:])

		if err == io.EOF {
			c <- string("")
			quit = true
		} else {
			c <- string(buff[0:numBytesReceived])
		}
	}
}
