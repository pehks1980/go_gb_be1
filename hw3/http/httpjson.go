package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	_ "path/filepath"
	"strings"
	"time"
)

type Employee struct {
	Name   string  `json:"name"`
	Age    int     `json:"age"`
	Salary float32 `json:"salary"`
}

type Handler struct {
}

type UploadHandler struct {
	UploadDir string
	HostAddr  string
}

// загрузка файлов
// curl -F 'file=@tst.txt' http://127.0.0.1/upload
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

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//http.ServeFile( w, r, "./index.html" )

	switch r.Method {
	case http.MethodGet:
		// get query key name ie: http://127.0.0.1:8080/?ext=jpeg
		ext := r.FormValue("ext")
		getdirlist(w, ext)
		// always reply to client
		fmt.Fprintf(w, "Parsed query-param with key \"ext\": %s", ext)
	case http.MethodPost:
		// get body of post :  curl 127.0.0.1 -X POST -d "dffdfdfd"
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Unable to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var employee Employee

		fmt.Fprintf(w, "Parsed request body: %s\n", string(body))

		err = json.Unmarshal(body, &employee)
		if err != nil {
			http.Error(w, "Unable to unmarshal JSON", http.StatusBadRequest)
			return
		}
		// curl 127.0.0.1 -X POST -d '{"name":"dfdff", "age":23, "salary":2345.0}'
		fmt.Fprintf(w, "Got a new employee!\nName: %s\nAge: %dy.o.\nSalary %0.2f\n",
			employee.Name,
			employee.Age,
			employee.Salary,
		)
	}
}

func getdirlist(w http.ResponseWriter, ext string) {

	htmlshapka := "<!DOCTYPE html>\n<html>\n<head>\n<style>\ntable {\n  font-family: arial, sans-serif;\n  " +
		"border-collapse: collapse;\n  width: 20%;\n}\n\ntd, th {\n  border: 1px solid #dddddd;\n  " +
		"text-align: left;\n  padding: 8px;\n}\n\ntr:nth-child(even) {\n  background-color: #dddddd;\n}\n</style>\n" +
		"</head>\n<body>\n\n<h2>List Directory</h2>"

	fmt.Fprint(w, htmlshapka)

	files, err := ioutil.ReadDir("./upload")
	if err != nil {
		log.Fatal(err)
	}

	htmlbegtable := "<table><tr>\n    <th>File</th>\n    <th>Ext</th>\n    <th>Size (b)</th>\n  </tr>\n"
	fmt.Fprint(w, htmlbegtable)
	for _, f := range files {
		Name := strings.Split(f.Name(), ".")
		if len(Name) > 1 {
			if Name[1] == ext || ext == "" {
				fmt.Fprintf(w, "<tr> <td>%s</td> <td>%s</td><td>%d</td></tr>\n", Name[0], Name[1], f.Size())
			}
		} else {
			if ext == "" {
				fmt.Fprintf(w, "<tr> <td>%s</td> <td>N/A</td><td>%d</td></tr>\n", Name[0], f.Size())
			}
		}

	}
	htmlendtable := "</table>\n\n</body>\n</html>"
	fmt.Fprint(w, htmlendtable)

}

func main() {
	//worldHandler := &helloHandler{"World"}
	//roomHandler := &helloHandler{"Mark"}
	handler := &Handler{}

	//http.Handle("/world", worldHandler)
	//http.Handle("/room", roomHandler)
	http.Handle("/", handler)

	srv := &http.Server{
		Addr:         ":80",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	uploadHandler := &UploadHandler{
		UploadDir: "upload",
		HostAddr:  "http://127.0.0.1/upload",
	}
	http.Handle("/upload", uploadHandler)

	go srv.ListenAndServe()

	// режим отображения папки через браузер

	fs := http.FileServer(http.Dir("./upload"))
	http.Handle("/upload/", http.StripPrefix("/upload/", fs))
	//http.Handle("/", fs)
	log.Println("Listening on :8080...")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}

	/*dirToServe := http.Dir(uploadHandler.UploadDir)
	fmt.Printf("DirTo Serve %s \n", dirToServe)
	fs := &http.Server{
		Addr:         ":8080",
		Handler:      http.FileServer(dirToServe),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	fs.ListenAndServe()*/

}
