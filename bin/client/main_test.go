package main

import (
	"io"
	"net"
	"testing"
	"time"
)

type AddrStub struct{}

func (a AddrStub) Network() string {
	return "test"
}

func (a AddrStub) String() string {
	return "local_unit_test"
}

type ConnMock struct {
	currMessageIdx int
	recvMessages   []string
	sendMessages   []string
	// Use pipe to block
	pipeRecvW *io.PipeWriter
}

func (c *ConnMock) Read(b []byte) (n int, err error) {
	n, err = baseRead(b, c.recvMessages, &c.currMessageIdx)

	// Blocks if empty
	if n == 0 {
		c.pipeRecvW.Write([]byte(""))
	}

	return n, err
}

func (c *ConnMock) Write(b []byte) (n int, err error) {
	n, err = baseWrite(b, &c.sendMessages)

	return n, err
}

func (c *ConnMock) Close() error {
	return nil
}

func (c *ConnMock) LocalAddr() net.Addr {
	return AddrStub{}
}

func (c *ConnMock) RemoteAddr() net.Addr {
	return AddrStub{}
}

func (c *ConnMock) SetDeadline(t time.Time) error {
	return nil
}

func (c *ConnMock) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *ConnMock) SetWriteDeadline(t time.Time) error {
	return nil
}

type StdoutMock struct {
	sendMessages []string
}

func (m *StdoutMock) Write(b []byte) (n int, err error) {
	n, err = baseWrite(b, &m.sendMessages)

	// Ignore added \n for convenience
	m.sendMessages[len(m.sendMessages)-1] = m.sendMessages[len(m.sendMessages)-1][:len(m.sendMessages[len(m.sendMessages)-1])-1]

	return n, err
}

type StdinMock struct {
	currMessageIdx int
	recvMessages   []string
}

func (m *StdinMock) Read(b []byte) (n int, err error) {
	n, err = baseRead(b, m.recvMessages, &m.currMessageIdx)

	return n, err
}

func baseRead(b []byte, recvMessages []string, currMessageIdx *int) (n int, err error) {
	if *currMessageIdx >= len(recvMessages) {
		return 0, nil
	}

	message := recvMessages[*currMessageIdx]
	(*currMessageIdx)++
	n = copy(b, []byte(message))

	return n, nil
}

func baseWrite(b []byte, sendMessages *[]string) (n int, err error) {
	message := string(b)
	n = len(message)

	*sendMessages = append(*sendMessages, message)

	return n, nil
}

type TestInput struct {
	connection *ConnMock
	stdin      *StdinMock
	stdout     *StdoutMock
}

func initTestInput(serverRecvMsg, serverSendMsg, stdinMsg, stdoutMsg []string) (t TestInput) {
	_, servRecvW := io.Pipe()

	t = TestInput{
		connection: &ConnMock{
			recvMessages: serverRecvMsg,
			sendMessages: serverSendMsg,
			pipeRecvW:    servRecvW,
		},

		stdin:  &StdinMock{recvMessages: stdinMsg},
		stdout: &StdoutMock{sendMessages: stdoutMsg},
	}

	return t
}

func TestHandleMessageFromServer(t *testing.T) {
	// Setup
	tests := map[string]struct {
		input               TestInput
		numExpectedMessages int
	}{
		"no message": {
			input:               initTestInput([]string{""}, []string{""}, []string{""}, []string{}),
			numExpectedMessages: 1,
		},
		"1 message": {
			input:               initTestInput([]string{"1msg"}, []string{""}, []string{""}, []string{}),
			numExpectedMessages: 2,
		},
		"multiple messages": {
			input:               initTestInput([]string{"1st", "2nd", "3rd"}, []string{""}, []string{""}, []string{}),
			numExpectedMessages: 4,
		},
		"multiple messages with end in between": {
			input:               initTestInput([]string{"1st", "2nd", "", "ignore"}, []string{""}, []string{""}, []string{}),
			numExpectedMessages: 3,
		},
	}

	for name, tt := range tests {
		tt := tt

		t.Run(name, func(t *testing.T) {
			handle(tt.input.connection, tt.input.stdin, tt.input.stdout)
			if tt.numExpectedMessages != len(tt.input.stdout.sendMessages) {
				t.Logf("Expected number of messages received that are written to the stdout do not match number of messages written to stdout -- Got: {%d} Expected: {%d}", len(tt.input.stdout.sendMessages), tt.numExpectedMessages)

				if len(tt.input.stdout.sendMessages) > 0 {
					t.Logf("Contents of stdout:")
					for _, v := range tt.input.stdout.sendMessages {
						t.Logf("{%s} with len %d", v, len(v))
					}
				}

				t.FailNow()
			}

			for i := 0; i < len(tt.input.connection.recvMessages); i++ {
				srvMessage := tt.input.connection.recvMessages[i]
				stdoutMessage := tt.input.stdout.sendMessages[i]
				if srvMessage != stdoutMessage && (srvMessage != "") {
					t.Errorf("Order of messages did not match -- Got: %s, Expected: %s", stdoutMessage, srvMessage)
				} else if i == len(tt.input.stdout.sendMessages)-1 {
					break
				}
			}
		})
	}
}

func TestHandleSendMessageToServer(t *testing.T) {
	// Setup
	tests := map[string]struct {
		input               TestInput
		numExpectedMessages int
	}{
		"no message": {
			input:               initTestInput([]string{}, []string{""}, []string{""}, []string{}),
			numExpectedMessages: 1,
		},
		"1 message": {
			input:               initTestInput([]string{}, []string{""}, []string{"1msg", ""}, []string{}),
			numExpectedMessages: 2,
		},
		"multiple messages": {
			input:               initTestInput([]string{}, []string{""}, []string{"1st", "2nd", "3rd", ""}, []string{}),
			numExpectedMessages: 4,
		},
	}

	for name, tt := range tests {
		tt := tt

		t.Run(name, func(t *testing.T) {
			handle(tt.input.connection, tt.input.stdin, tt.input.stdout)
			if tt.numExpectedMessages != len(tt.input.stdin.recvMessages) {
				t.Logf("Expected number of messages received that are written to the server do not match number of messages written to stdin -- Got: {%d} Expected: {%d}", len(tt.input.stdin.recvMessages), tt.numExpectedMessages)

				if len(tt.input.stdin.recvMessages) > 0 {
					t.Logf("Contents of stdin:")
					for _, v := range tt.input.stdin.recvMessages {
						t.Logf("{%s} with len %d", v, len(v))
					}
				}

				t.FailNow()
			}

			for i := 0; i < len(tt.input.connection.recvMessages); i++ {
				srvMessage := tt.input.connection.recvMessages[i]
				stdinMessage := tt.input.stdin.recvMessages[i]
				if srvMessage != stdinMessage && (srvMessage != "") {
					t.Errorf("Order of messages did not match -- Got: %s, Expected: %s", stdinMessage, srvMessage)
				} else if i == len(tt.input.stdin.recvMessages)-1 {
					break
				}
			}
		})
	}
}

func TestHandleServerEchoReceived(t *testing.T) {
	// Setup
	tests := map[string]struct {
		input               TestInput
		numExpectedMessages int
	}{
		"no message": {
			input:               initTestInput([]string{}, []string{}, []string{""}, []string{}),
			numExpectedMessages: 0,
		},
		"1 message": {
			input:               initTestInput([]string{}, []string{}, []string{"1msg\n", ""}, []string{}),
			numExpectedMessages: 1,
		},
		"multiple messages": {
			input:               initTestInput([]string{}, []string{}, []string{"1st\n", "2nd\n", "3rd\n", ""}, []string{}),
			numExpectedMessages: 3,
		},
	}

	for name, tt := range tests {
		tt := tt

		t.Run(name, func(t *testing.T) {
			handle(tt.input.connection, tt.input.stdin, tt.input.stdout)

			if tt.numExpectedMessages != len(tt.input.connection.sendMessages) {
				t.Logf("Expected number of messages sent do not match number of messages written to stdout -- Got: {%d} Expected: {%d}", len(tt.input.connection.sendMessages), tt.numExpectedMessages)

				if len(tt.input.stdin.recvMessages) > 0 {
					t.Logf("Contents of stdin:")
					for _, v := range tt.input.stdin.recvMessages {
						t.Logf("{%s} with len %d", v, len(v))
					}
				}

				if len(tt.input.connection.sendMessages) > 0 {
					t.Logf("Contents of connection sent messages:")
					for _, v := range tt.input.connection.sendMessages {
						t.Logf("{%s} with len %d", v, len(v))
					}
				}

				t.FailNow()
			}

			for i := 0; i < len(tt.input.connection.sendMessages); i++ {
				srvSentMessage := tt.input.connection.sendMessages[i]
				stdinMessage := tt.input.stdin.recvMessages[i]

				// Ignore added \n for convenience
				stdinMessage = stdinMessage[:len(stdinMessage)-1]

				if srvSentMessage != stdinMessage && (srvSentMessage != "") {
					t.Errorf("Order of messages did not match -- Got: %s, Expected: %s", stdinMessage, srvSentMessage)
				} else if i == len(tt.input.stdin.recvMessages)-1 {
					break
				}
			}
		})
	}
}
