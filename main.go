package main

import (
	"fmt"
	"github.com/opentracing/opentracing-go"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"testOpentracing/pkg/tracing"
	"time"
)

const (
	host  = "127.0.0.1"
	bPort = 39121
	cPort = 39122
)

// write string to connection
func handleWrite(conn net.Conn, input string, done chan string, isEnd bool) {
	fmt.Println("write: ", input)
	_, e := conn.Write([]byte(input+ "\n"))
	if e != nil {
		fmt.Println("Error to send message because of ", e.Error())
	}
	if isEnd {
		_, e = conn.Write([]byte("EOF" + "\n"))
		fmt.Println("write EOF!")
		if e != nil {
			fmt.Println("Error to send message because of ", e.Error())
		}
	}
	done <- ""
}

// read string from connection
func handleRead(conn net.Conn, done chan string) {
	for {
		buf := make([]byte, 2048)
		reqLen, err := conn.Read(buf)
		if err != nil {
			fmt.Println("Error to read message because of ", err)
			return
		}
		res := string(buf[:reqLen-1])
		strs := strings.Split(res, "\n")
		for _, str := range strs {
			if str == "EOF" {
				close(done)
				return
			}
			fmt.Println("read from socket: ", str)
			done <- str
		}
	}
}

func askB(spanID string, input int, output chan string) {
	addr := host + ":" + strconv.Itoa(bPort)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Println("error", err)
	}
	defer conn.Close()
	done := make(chan string)
	go handleWrite(conn, spanID, done, false)
	<-done
	go handleWrite(conn, strconv.Itoa(input), done, true)
	<-done
	go handleRead(conn, done)
	res := ""
	for ans := range done {
		res = ans
		output <- res
	}
	close(output)
}

func askC(spanID string, input int, output chan string) {
	addr := host + ":" + strconv.Itoa(cPort)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Println("error", err)
	}
	defer conn.Close()
	done := make(chan string)
	go handleWrite(conn, spanID, done, false)
	<-done
	go handleWrite(conn, strconv.Itoa(input), done, true)
	<-done
	go handleRead(conn, done)
	res := ""
	for ans := range done {
		res = ans
		output <- res
	}
	close(output)
}

func handleRequest(conn net.Conn, port int) {
	defer conn.Close()
	ch := make(chan string)
	go handleRead(conn, ch)
	parentRequestID := <-ch
	operationName := "func: handleRequest, server: "
	if port == bPort {
		operationName += "b"
	}else {
		operationName += "c"
	}
	span := tracing.CreateChildFromCarrier(operationName, parentRequestID)
	defer span.Finish()
	requestID := tracing.GetCarrier(span)
	output := 0
	if port == cPort {
		span.SetTag("server", "c")
		input, err := strconv.Atoi(<-ch)
		if err != nil {
			fmt.Println("error: ", err)
		}
		span.LogKV("input", input)
		output = input - 1
		span.LogKV("output", output)
	} else {
		span.SetTag("server", "b")
		input, err := strconv.Atoi(<-ch)
		if err != nil {
			fmt.Println("error: ", err)
		}
		span.LogKV("input", input)
		for ; input > 0; {
			outCh := make(chan string)
			output += input
			go askC(requestID, input, outCh)
			<-outCh // request id
			input, err = strconv.Atoi(<-outCh )
		}
		span.LogKV("output", output)
	}
	done := make(chan string)
	go handleWrite(conn, requestID, done, false)
	<-done
	go handleWrite(conn, strconv.Itoa(output), done, true)
	<-done
}

func startSocket(port int) {
	addr := host + ":" + strconv.Itoa(port)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Println("Error listening:", err)
		os.Exit(1)
	}
	defer l.Close()
	fmt.Println("Listening on " + addr)
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err)
			os.Exit(1)
		}
		//logs an incoming message
		fmt.Printf("Received message %s -> %s \n", conn.RemoteAddr(), conn.LocalAddr())
		// Handle connections in a new goroutine.
		go handleRequest(conn, port)
	}
}

func main() {
	rand.Seed(int64(time.Now().UnixNano()))
	serviceName := "service "
	switch hello := os.Args[1]; hello {
	case "b":
		serviceName += "b: sum(k) from 1 to k"
	case "c":
		serviceName += "c: next(k) return k -1"
	case "a":
		serviceName += "a: invoke service b"
	}
	closer, err := tracing.InitTracer(serviceName, "localhost:5776", true)
	//closer, err := tracing.InitTracerFromYAML("./tracer.yaml")
	if err != nil {
		fmt.Println("error", err)
	}
	defer closer.Close()

	switch hello := os.Args[1]; hello {
	case "b":
		port := bPort
		startSocket(port)
	case "c":
		port := cPort
		startSocket(port)
	case "a":
		output := make(chan string)
		input, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Println("error", err)
		}
		HandleAddService(output, input)
	}
}

func HandleAddService(output chan string, input int)  {
	span := tracing.CreateSpan(fmt.Sprintf("HandleAddService input=%d", input))
	defer span.Finish()
	requestID :=tracing.GetCarrier(span)

	DoHandleStep1(span)

	fmt.Println("input: ", input)
	span.LogKV("input", input)
	go askB(requestID, input, output)
	//go askC(requestID, input, output)
	res1 := <-output
	result := <-output
	span.LogKV("output", result)
	fmt.Println("request id : ", res1, "  result: ", result)
}

func DoHandleStep1(span opentracing.Span)  {
	tracer := opentracing.GlobalTracer()
	childSpan := tracer.StartSpan("Do handle Step 1", opentracing.ChildOf(span.Context()))
	defer childSpan.Finish()
}
