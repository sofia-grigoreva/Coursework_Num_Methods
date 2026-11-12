package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

const resultsDir = "output"

func main() {
	dataset := flag.String("dataset", "YorkUrbanDB", "каталог исходных фотографий")
	output := flag.String("output", "build/report", "каталог сайта")
	host := flag.String("host", "127.0.0.1", "адрес HTTP-сервера")
	port := flag.Int("port", 2107, "порт HTTP-сервера")
	serve := flag.Bool("serve", true, "запустить сервер после сборки сайта")
	flag.Parse()

	examples, err := loadExamples(resultsDir)
	if err != nil {
		log.Fatal(err)
	}
	quality, err := loadQuality(*dataset)
	if err != nil {
		log.Fatal(err)
	}
	if err := writeSite(*output, examples, quality); err != nil {
		log.Fatal(err)
	}
	if !*serve {
		return
	}

	photos, err := os.OpenRoot(*dataset)
	if err != nil {
		log.Fatal(err)
	}
	defer photos.Close()
	results, err := os.OpenRoot(resultsDir)
	if err != nil {
		log.Fatal(err)
	}
	defer results.Close()

	address := net.JoinHostPort(*host, strconv.Itoa(*port))
	server := &http.Server{
		Addr:              address,
		Handler:           handler(*output, photos.FS(), results.FS(), examples),
		ReadHeaderTimeout: 10 * time.Second,
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Сервер запущен на http://%s\n", listener.Addr())
	log.Fatal(server.Serve(listener))
}
