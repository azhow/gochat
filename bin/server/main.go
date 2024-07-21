package main

import (
	"fmt"
	"gocc/gochat/pkg/utils"
	"io"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", utils.IpAddr+":"+utils.SrvPort)
	var connArray []net.Conn
	c := make(chan string)

	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	defer listener.Close()

	fmt.Println("Listening to connections on port 7007")

	go echoMessages(c, &connArray)

	for {
		connection, err := listener.Accept()

		if err != nil {
			fmt.Println("Error accepting connection: ", err)
		}
		connArray = append(connArray, connection)

		fmt.Println("Accepted connection: ", connection.RemoteAddr())
		go handleClient(connection, c)
	}
}

func handleClient(conn net.Conn, c chan string) {
	defer conn.Close()

	var name string
	buff := make([]byte, 1024)

	_, _ = conn.Write([]byte("Name: "))
	numBytesRead, _ := conn.Read(buff[0:])
	name = string(buff[0:numBytesRead])

	readIntoChannel(name, conn, c)
}

func echoMessages(c chan string, connArray *[]net.Conn) {
	for {
		select {
		case message := <-c:
			for _, conn := range *connArray {
				_, _ = conn.Write([]byte(message))
			}
		}
	}
}

func readIntoChannel(username string, conn net.Conn, c chan string) {
	quit := false
	for !quit {
		buff := make([]byte, 1024)

		numBytesRead, err := conn.Read(buff[0:])

		if err == io.EOF {
			c <- string("")
			quit = true
		} else {
			message := string(buff[0:numBytesRead])
			c <- username + ": " + message
		}
	}
}
