package main_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type Handler struct {

}

type Employee struct {
	Name   string  `json:"name"`
	Age    int     `json:"age"`
	Salary float32 `json:"salary"`
}

// тестируемая вьюха
func (h Handler) ServeHTTP(w *httptest.ResponseRecorder, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// get query key name ie: 127.0.0.1/?name=blabla
		name := r.FormValue("name")
		// always reply to client
		fmt.Fprintf(w, "Parsed query-param with key \"name\": %s", name)
	case http.MethodPost:
		// get body of post :  curl 127.0.0.1 -X POST -d "dffdfdfd"
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Unable to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var employee Employee

		fmt.Printf("Parsed request body: %s\n",string(body))

		err = json.Unmarshal(body, &employee)
		if err != nil {
			http.Error(w, "Unable to unmarshal JSON", http.StatusBadRequest)
			return
		}
		// curl 127.0.0.1 -X POST -d '{"name":"dfdff", "age":23, "salary":2345.0}'
		fmt.Fprintf(w, "Got a new employee! Name: %s Age: %d Salary %0.2f",
			employee.Name,
			employee.Age,
			employee.Salary,
		)
	}
}

type UploadHandler struct {
	UploadDir string
	HostAddr string
}

func (u UploadHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	// get file from request
	file, fileheader, err := request.FormFile("file")
	if err != nil {
		http.Error(writer, "Unable to read file", http.StatusBadRequest)
		return
	}
	defer file.Close()
	fmt.Println(fileheader)
	filebody, err := ioutil.ReadAll(file)
	if err != nil {
		http.Error(writer, "Unable to read file", http.StatusBadRequest)
		return
	}
	// u.UploadDir - из структуры обработчика его текущая папка
	//u.UploadDir = "upl" // можно задать какой хочешь, по умолчанию тот что в пути урла(и  в хендлере)

	files, err := ioutil.ReadDir("./upload")
	if err != nil {
		log.Fatal(err)
	}

	var filesnames []string

	for _, fi := range files {
		filesnames = append(filesnames, fi.Name())
	}
	//get unique name
	newfilename := getuniquefilename(filesnames, fileheader.Filename)

	filePath := u.UploadDir + "/" + newfilename

	err = ioutil.WriteFile(filePath, filebody, 0777)
	if err != nil {
		log.Println(err)
		http.Error(writer, "Unable to save file", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(writer, "File %s has been successfully uploaded\n", newfilename)

	fileurl := u.HostAddr + "/" + newfilename

	fmt.Fprintf(writer, "File URL: %s\n", fileurl)
}

func getuniquefilename(filesnames []string, filename string) string {
	unifilename := filename
	for _, f := range filesnames {
		if filename == f {
			Name := strings.Split(f, ".")
			newfilename := Name[0] + "_" + "1" + "." + Name[1]
			return getuniquefilename(filesnames, newfilename)
		}
	}
	return unifilename
}

// get тест из методы
func TestGetHandler(t *testing.T) {
	// Создаем запрос с указанием нашего хендлера. Так как мы тестируем GET-эндпоинт
	// то нам не нужно передавать тело, поэтому третьим аргументом передаем nil
	req, err := http.NewRequest("GET", "/?name=John", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Мы создаем ResponseRecorder(реализует интерфейс http.ResponseWriter)
	// и используем его для получения ответа
	rr := httptest.NewRecorder()
	handler := &Handler{}

	// Наш хендлер соответствует интерфейсу http.Handler, а значит
	// мы можем использовать ServeHTTP и напрямую указать
	// Request и ResponseRecorder
	handler.ServeHTTP(rr, req)

	// Проверяем статус-код ответа
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Проверяем тело ответа
	expected := `Parsed query-param with key "name": John`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

// пост тест из методы
func TestPostHandler(t *testing.T) {
	// Создаем запрос с указанием нашего хендлера. Так как мы тестируем GET-эндпоинт
	// то нам не нужно передавать тело, поэтому третьим аргументом передаем nil
	testbody := "{\"name\":\"dfdff\", \"age\":23, \"salary\":2345.0}"
	//body := &bytes.Buffer{}
	body := strings.NewReader(testbody)
	req, err := http.NewRequest("POST", "/", body)
	if err != nil {
		t.Fatal(err)
	}

	// Мы создаем ResponseRecorder(реализует интерфейс http.ResponseWriter)
	// и используем его для получения ответа
	rr := httptest.NewRecorder()
	handler := &Handler{}

	// Наш хендлер соответствует интерфейсу http.Handler, а значит
	// мы можем использовать ServeHTTP и напрямую указать
	// Request и ResponseRecorder
	handler.ServeHTTP(rr, req)

	// Проверяем статус-код ответа
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Проверяем тело ответа
	expected := "Got a new employee! Name: dfdff Age: 23 Salary 2345.00"
	resulted := rr.Body.String()
	if resulted != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

// открывает файл на диске, ложит его в пост запрос заголовок все такое
// req - запрос с файлом rr тестовый рекордер он принимает все что выдает тестируемый обработчик
// мы вызываем функцию с подложной своей хендлер стуктурой на входе реквест на выходе запись в rr тестовый
// upload test адаптированный к модифицированному обработчику загрузки файлов на сервер
func TestUploadHandler(t *testing.T) {
	// remove test file from upload
	os.Remove("./upload/index_1.html")
	// открываем файл, который хотим отправить
	file, _ := os.Open("index.html")
	defer file.Close()

	// действия, необходимые для того, чтобы засунуть файл в запрос
	// в качестве мультипарт-формы
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", filepath.Base(file.Name()))
	io.Copy(part, file)
	writer.Close()

	// опять создаем запрос, теперь уже на /upload эндпоинт
	req, _ := http.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Add("Content-Type", writer.FormDataContentType())

	// создаем ResponseRecorder
	rr := httptest.NewRecorder()

	// создаем заглушку файлового сервера. Для прохождения тестов
	// нам достаточно чтобы он возвращал 200 статус
	// мок - cервер для теста принимает и отдает 200 ок
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok!")
	}))
	defer ts.Close()
    // структура натравливает метод ServHTTP оригинальный на работу с этим фейковым сервером ts
	uploadHandler := &UploadHandler{
		UploadDir: "upload",
		// таким образом мы подменим адрес файлового сервера
		// и вместо реального, хэндлер будет стучаться на заглушку
		// которая всегда будет возвращать 200 статус, что нам и нужна
		HostAddr:  ts.URL,
	}

	// опять же, вызываем ServeHTTP у тестируемого обработчика
	uploadHandler.ServeHTTP(rr, req)

	// Проверяем статус-код ответа
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	wantfile := "index_1.html"

	expected := "File "+ wantfile + " has been successfully uploaded\n" +
        "File URL: " + ts.URL + "/" + wantfile

	if !strings.Contains(rr.Body.String(), expected) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}
