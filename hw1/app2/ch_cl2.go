
package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

func main() {
	// коннект к серверу
	conn, err := net.Dial("tcp", "localhost:8002")
	if err != nil {
		log.Fatal(err)
	}
	// обработчик при закрытии коннекта
	defer func(){
		err := conn.Close()
		log.Printf(" error when closing conn: %v",err)
	}()

	//defer conn.Close()

	// горутина принимает что пришло по сет коннекту и кидает на std out
	go func() {
		io.Copy(os.Stdout, conn)
	}()
	// кидает с клавы на коннект
	io.Copy(conn, os.Stdin) // until you send ^Z
	fmt.Printf("%s: exit", conn.LocalAddr())
}
