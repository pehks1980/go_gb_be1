package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"strconv"
	"strings"
)

/*

2. Доработка приложения чата - done
Добавить в приложение чата возможность устанавливать клиентам свой никнейм при подключении к серверу.
Как это и бывает в чатах, никнеймы, заданные клиентами, должны отображаться слева от отправленных ими сообщений.

* 3. Математика на скорость
Дополнительное задание

Реализовать игру “Математика на скорость”: сервер генерирует случайное выражение с двумя операндами,
сохраняет ответ, а затем отправляет выражение всем клиентам.
Первый клиент, отправивший правильный ответ - побеждает, затем генерируется следующее выражение и так далее. - done
*/

import (
	"bufio"
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// сущность клиента канал
type client chan<- string // типчик client это канал с write only те только пушить в него
/*
chan   // read-write
<-chan // read only
chan<- // write only
*/
type clientStruc struct {
	id      string
	nick    string
	channel client
}

var (
	entering = make(chan clientStruc)
	leaving  = make(chan clientStruc)
	messages = make(chan string)
)

func main() {
	listener, err := net.Listen("tcp", "localhost:8002")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("server is accepting connections on %s \n старт игры - ввести что нить и нажать ентер\n", listener.Addr())
	ctx, cancel := context.WithCancel(context.Background())
    // Обр сигналов
	go catchOs(cancel)
    // Инит игроструктуры
	game := NewMathGame(20)
	// рассыльшик
	go broadcaster(ctx, game)
	// обработчик игры
	go mathgame(ctx, game)

	newconn := true

	for {
		select {
		case <-ctx.Done():
			// управление - досрочное завершения
			fmt.Printf("server is shut down!!! in 5 seconds...\n")
			//ждем 5 с пока сервер в горутинках клинтов их не поодключает
			time.Sleep(5 * time.Second)
			return

		default:
			if newconn {
				// горутинка принимает коннекшены и запускает их обработчики
				go func() {
					newconn = false
					conn, err := listener.Accept()
					if err != nil {
						log.Print(err)
					} else {
						// запуск под каждый успешный коннект клиента обработчика - он его обслуживает пока тот не покинет чат
						go handleConn1(conn, ctx)
					}
					newconn = true
				}()
			}
		}
	}
}

// MathGame Cтруктуры игры (используется пример сложения 2 чисел)
type MathGame struct {
	op1, op2, res int
	gamemode      bool
}

func NewMathGame(maxRange int) *MathGame {
	op1 := rand.Intn(maxRange)
	op2 := rand.Intn(maxRange)
	return &MathGame{
		op1: op1,
		op2: op2,
		res: op1 + op2,
	}
}

func (MathGame *MathGame) GenMathGame(maxRange int) {
	//rand.Seed(12345)
	MathGame.op1 = rand.Intn(maxRange)
	MathGame.op2 = rand.Intn(maxRange)
	MathGame.res = MathGame.op1 + MathGame.op2
	MathGame.gamemode = true
}

func (MathGame *MathGame) CheckAnswer(answer string) bool {
	if s, err := strconv.Atoi(answer); err == nil {
		// проверка на правильный ответ
		if s == MathGame.res {
			return true
		}
	}
	return false
}

// правильны ответ должен быть в канале
func mathgame(ctx context.Context, game *MathGame) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		// старт игры - ввести что нить и нажать ентер
		game.GenMathGame(20)
		messages <- fmt.Sprintf("0:0:Решите пример - %d + %d =?", game.op1, game.op2)
	}
}

// ловим сигналы выключения сервера
func catchOs(cancel context.CancelFunc) {
	osSignalChan := make(chan os.Signal)

	signal.Notify(osSignalChan, syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGUSR1)
	for {
		// поточек ждет сигнал в канале
		select {
		case sig := <-osSignalChan:
			log.Printf("got %s signal", sig.String())
			switch sig.String() {
			case "interrupt":
				cancel()
				return
			case "quit":
				cancel()
				return

			}

		}
	}

}

// обработчик сообщений горутин чата
func broadcaster(ctx context.Context, game *MathGame) {
	// мапа базы клиентов
	clients := make(map[clientStruc]bool)
	shutannounce := true
	for {
		select {
		case <-ctx.Done():
			// управление - досрочное завершения оповещение (1 раз)
			if shutannounce {
				for cli := range clients {
					cli.channel <- fmt.Sprintf("server is shut down!!! in 5 seconds...\n")
				}
				shutannounce = false
			}

		//  пришло сообщение
		case msg := <-messages:
			// отправка в каналы клиентов этого сообщения
			fmt.Println(msg)
			// check id in msg
			// find nick
			// add it to msg and broadcast it
			tokens := strings.Split(msg, ":")
			// find nick by id (sorry no map)

			nick := "Server"
			var nickchan client
			nickchan = nil
			// поиск ника в бд клиентов
			for cli := range clients {
				id := strings.Split(cli.id, ":")
				if id[0] == tokens[0] && id[1] == tokens[1] {
					nick = cli.nick
					nickchan = cli.channel
					break
				}
			}
			// проверка ответа клиента икс в режиме игры
			if game.gamemode && nick != "Server" {
				// ответ в token[2]
				if game.CheckAnswer(tokens[2]) {
					for cli := range clients {
						cli.channel <- fmt.Sprintf("%s: первый ответил, правильный ответ был - %s\nИгра окончена \n", nick, tokens[2])
					}
					game.gamemode = false
				} else {
					nickchan <- fmt.Sprintf("%s: неправильный ответ, попробуйте еще раз...\n", nick)
				}

			} else {
				// рассылка Обычного сообщения с этим ником
				for cli := range clients {
					cli.channel <- fmt.Sprintf("%s: %s\n", nick, tokens[2])
				}
			}

		// зашел какой то чел канал икс - cli
		case cli := <-entering:
			// регистрация cli в "базе клиентов" (cli это канал клиента икс)
			clients[cli] = true
		// сделала больно и покинула чат
		case cli := <-leaving:
			// лог аут удаление клиента из "базы клиентов"
			delete(clients, cli)
			close(cli.channel)
		}
	}
}

// горутина обработчика клиента икс
func handleConn1(conn net.Conn, ctx context.Context) {

	// канал сообщений от сервера к клиенту икс
	ch := make(chan string)
	// по сути питоний генератор - при отправке в ch cообщения произойдет автоматическая его отпавка клиенту
	go clientWriter(conn, ch)
	// текстовый ID = 'IP:PORT'
	id := conn.RemoteAddr().String()
	// пишем клиенту запрос задать ник
	ch <- "Please enter your Nick. Like - Nick:Nickname<enter>\n"
	// в цикле все строки (раздел по энтеру) - что приходит от клиента кидается в канал бродаст мессаджей

	expectedInput := true
	clientstruc := clientStruc{}
loop:
	for {
		select {
		case <-ctx.Done():
			// управление - досрочное завершения
			fmt.Printf("cancel conn %s\n", conn.RemoteAddr())
			break loop

		default:
			if expectedInput {
				expectedInput = false
				// инлайн горутинка принимает мессаджи от клиента по одному за раз
				go func() {
					scanner := bufio.NewScanner(conn)
					scanner.Scan()

					// разборка сообщения - нп предмет спец сообщения задачи ника
					// message special Nick:bla-bla will set Nick as bla-bla
					tokens := strings.Split(scanner.Text(), ":")
					if tokens[0] == "Nick" {
						// регим клинта:
						// кидаем в канал структуру с данными по этому клиенту (для занесения его в базу данных клиентов)
						clientstruc = clientStruc{channel: ch,
							id: id, nick: tokens[1]} //id "IP:PORT"
						entering <- clientstruc
						ch <- "Your Nick is accepted.\n"
						messages <- clientstruc.id + ":" + tokens[1] + " has arrived"
					} else {
						// форматир ID:сообщение от клиента
						// рассылка всем клиентам в таком формате
						messages <- clientstruc.id + ":" + scanner.Text()
					}
					//по завершению разрешаем обработку след. сообщения от клиента икс
					expectedInput = true
				}()
			}

		}

	}
	// по выходу из цикла контекст сработал
	// обработчик закрытия соединения клиента икс
	// канал отправляем стуктуру клиента на вычеркивание из базы данных клиентов
	messages <- clientstruc.id + ":" + clientstruc.nick + " has been disconnected \n"
	leaving <- clientstruc
	err := conn.Close()
	if err != nil {
		log.Printf(" error when closing conn: %v", err)
	}
}

// write генератор в сокет клиенту
func clientWriter(conn net.Conn, ch <-chan string) {
	for msg := range ch {
		fmt.Fprintf(conn, msg)
	}
}
