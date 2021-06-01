package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:8000")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	fmt.Printf("Connected to %s. \n ...Hit any key to stop...",conn.RemoteAddr())
	// горутина принимает что пришло по сет коннекту и кидает на std out
	go func() {
		scanner := bufio.NewScanner(conn)
		for scanner.Scan(){
			// форматир ID:сообщение от клиента
			fmt.Println(scanner.Text())
		}
	}()

	b := ""
	fmt.Scan(&b)
/*
	buf := make([]byte, 256) // создаем буфер
	for {
		_, err = conn.Read(buf)
		if err == io.EOF {
			break
		}
		io.WriteString(os.Stdout, fmt.Sprintf("Custom output! %s", string(buf))) // выводим измененное сообщение сервера в консоль
	}

 */
}


