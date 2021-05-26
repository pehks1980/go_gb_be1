package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
)

/*
1. Доработка приложения рассылки даты/времени - Done
Добавить в приложение рассылки даты/времени возможность отправлять клиентам
произвольные сообщения из консоли сервера.
Сообщения должны работать по принципу броадкастинга, т. е.
должны отправляться всем подключенным клиентам.
*/
type clientCh chan<- string // типчик client это канал с write only те только пушить в него

var (
	regclient = make(chan clientCh)
	message = make(chan string)
)

func main() {
	listener, err := net.Listen("tcp", "localhost:8000")
	if err != nil {
		log.Fatal(err)
	}

	go broadcaster1()

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan(){
			// форматир ID:сообщение от клиента
			message <- "Python Kaa says: " + scanner.Text()
		}
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		handleConn(conn)
	}
}

// обработчик сообщений горутин чата
func broadcaster1() {
	// мапа базы клиентов
	clients := make(map[clientCh]bool)
	for {
		select {
		//  пришло сообщение
		case msg := <-message:
			// отправка в каналы клиентов этого сообщения
			fmt.Println(msg)
			for cli := range clients {
				cli <- fmt.Sprintf("%s\n", msg)

			}
		// зашел какой то чел канал икс - cli
		case cli := <- regclient:
			// регистрация cli в "базе клиентов" (cli это канал клиента икс)
			clients[cli] = true
		}

	}
}

func handleConn(c net.Conn) {
	defer c.Close()

	ch := make(chan string)

	regclient <- ch

	go func() {
		for msg := range ch {
			_, err := fmt.Fprintf(c, msg)
			if err != nil {
				return
			}
		}
	}()

	for {
		_, err := io.WriteString(c, time.Now().Format("server 15:04:05\n\r"))
		if err != nil {
			return
		}
		time.Sleep(1 * time.Second)
	}


}

