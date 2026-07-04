package httputils

import (
	"bufio"
	"bytes"
	"fmt"
	"testing"
)

func TestGetRandomUA(t *testing.T) {
	fmt.Println(GetRandomUA())
	fmt.Println(GetRandomUA())
}

func TestReadResponse(t *testing.T) {
	var a = "HTTP/1.1 200 OK\ndate: Mon, 09 Dec 2024 12:27:19 GMT\ncontent-type: text/html;charset=UTF-8\nTransfer-Encoding: chunked\nconnection: keep-alive\ncontent-language: zh-CN\r\n\r\n<html XXL-JOBstyle=\"height: auto; min-height: 100%;\"><head>"
	_, err := ReadResponse(bufio.NewReader(bytes.NewReader([]byte(a))))
	if err != nil {
		return
	}
}
