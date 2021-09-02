package main

import (
	"encoding/binary"
	"fmt"
	"github.com/forestgiant/sliceutil"
	"github.com/gorilla/mux"
	"github.com/namsral/flag"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var rootPath string
var clientHostsArg string
var clientHosts []string
var listenAddress string
var externalAddress string
var pollInterval int

type clientappSettings struct {
	name string
	rcvPort string
	extensions []string
}

func clientApps() map[string]clientappSettings {
	return map[string]clientappSettings{
		"Tinfoil": clientappSettings{
			name: "Tinfoil",
			rcvPort: "2000",
			extensions: []string{".nsp", ".nsz", ".xci"}},
		"FBI": clientappSettings{
			name: "FBI",
			rcvPort: "5000",
			extensions: []string{".cia", ".tik"}},
	}
}


func readArgs() {
	flagset := flag.NewFlagSetWithEnvPrefix(os.Args[0], "GOFOIL", 0)

	flagset.StringVar(&rootPath, "root", "/games", "Root path for files to serve")
	flagset.StringVar(&clientHostsArg, "clienthosts", "localhost",
					  "Comma-separated addresses for hosts to poll for Tinfoil or FBI.")
	flagset.StringVar(&listenAddress, "listenaddress", "0.0.0.0:8000", "IP address to bind server to.")
	flagset.StringVar(&externalAddress, "externaladdress", "0.0.0.0:8000",
					  "Address and port for clients to connect to the server under.")
	flagset.IntVar(&pollInterval, "pollinterval", 2, "How often to poll the target hosts.")
	flagset.Parse(os.Args[1:])
	clientHosts = strings.Split(clientHostsArg, ",")
}


func main() {

	readArgs()
	r := mux.NewRouter()
	r.HandleFunc("/", HealthcheckHandler)
	r.PathPrefix("/files/").Handler(http.StripPrefix("/files/", http.FileServer(http.Dir(rootPath))))

	log.Printf("Starting server at %s, external address %s.", listenAddress, externalAddress)
	log.Printf("Games root path: %s", rootPath)
	log.Printf("Hosts to poll: %s", clientHostsArg)

	srv := &http.Server{
		Handler: r,
		Addr: listenAddress
	}

	for _, clientHostname := range targetHosts {
		go pollHost(clientHostname)
	}

	err := srv.ListenAndServe()
	if err != nil {
		log.Panic(err)
	}
}

func HealthcheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusCreated)
}

func pollHost(targetHost string) {
	poll := time.Tick(time.Second * time.Duration(pollInterval))
	for _ = range poll {
		for name, settings := range clientApps() {
			conn, err := net.DialTimeout("tcp", targetHost + ":" + settings.rcvPort,
										 time.Second * time.Duration(pollInterval - 1))
			if err == nil {
				log.Printf("%s: found %s at %s", targetHost, name, conn.RemoteAddr())
				defer conn.Close()
				files, length := getFileList(rootPath, settings.extensions)
				sendFileList(conn, files, length)
			}
		}
	}
}

func getFileList(rootPath string, extensions []string) ([]string, int) {

	files := []string{}
	length := 0

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Failed to process %s: %s", path, err)
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if sliceutil.Contains(extensions, filepath.Ext(info.Name())) {
			relPath := strings.TrimPrefix(path, rootPath+"/")
			fileURL := fmt.Sprintf("%s/files/%s\n", externalAddress, url.PathEscape(relPath))
			files = append(files, fileURL)
			length += len(fileURL)
		}
		return nil
	})
	if err != nil {
		log.Printf("could not scan folders: %v", err)
		return nil, 0
	}

	sort.Strings(files)
	return files, length
}

func sendFileList(out io.Writer, fileList []string, fileListLength int) {
	// Taken from : https://github.com/bycEEE/tinfoilusbgo/blob/master/main.go. Adapted to work for network (bigEndian)
	// Start sending with length of file list in bytes
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(fileListLength))
	out.Write(buf)

	// Then send the file list line by line
	for _, path := range fileList {
		buf = make([]byte, len(path))
		copy(buf, path)
		out.Write(buf)
	}
}
