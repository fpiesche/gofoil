package main

import (
	"encoding/binary"
	"github.com/namsral/flag"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var rootPath string
var switchHostsArg string
var switchHosts []string
var hostIP string
var hostPort string
var externalHost string
var externalPort string
var pollInterval int


func readArgs() {
	flagset := flag.NewFlagSetWithEnvPrefix(os.Args[0], "GOFOIL", 0)

	flagset.StringVar(&rootPath, "root", "/games", "Root path for files to serve")
	flagset.StringVar(&switchHostsArg, "switchhosts", "localhost", "IP addresses or host names for Switches to check.")
	flagset.StringVar(&hostIP, "ip", "0.0.0.0", "IP address to bind server to.")
	flagset.StringVar(&hostPort, "port", "8000", "Port to open http server on.")
	flagset.StringVar(&externalHost, "externalhost", "0.0.0.0", "External IP address or host name to create download links on.")
	flagset.StringVar(&externalPort, "externalport", "8000", "The port the web server can be reached from the outside at.")
	flagset.IntVar(&pollInterval, "pollinterval", 2, "How often to poll for the presence of Switches.")
	flagset.Parse(os.Args[1:])
	switchHosts = strings.Split(switchHostsArg, ",")
}


func main() {

	readArgs()
	r := mux.NewRouter()
	r.HandleFunc("/", HealthcheckHandler)
	r.PathPrefix("/files/").Handler(http.StripPrefix("/files/", http.FileServer(http.Dir(rootPath))))

	log.Printf("Starting server at %s:%s.", hostIP, hostPort)
	log.Printf("Games root path: %s", rootPath)
	log.Printf("Switch hosts to poll: %s", switchHostsArg)

	srv := &http.Server{
		Handler: r,
		Addr:    hostIP + ":" + hostPort,
		// Good practice: enforce timeouts for servers you create!
		// But here the switch will be downloading file, slowly...
		//WriteTimeout: 15 * time.Second,
		//ReadTimeout:  15 * time.Second,
	}

	go pollForSwitches()

	err := srv.ListenAndServe()
	if err != nil {
		log.Panic(err)
	}
}

func HealthcheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusCreated)
}

func pollForSwitches() {
	poll := time.Tick(time.Second * time.Duration(pollInterval))
    for _ = range poll {
		for _, switchHost := range switchHosts {
			conn, err := net.Dial("tcp", switchHost + ":2000")
			if err == nil {
				log.Printf("Connected to Switch at %s", switchHost)
				defer conn.Close()
				sendFileList(conn)
			}
		}
	}
}

func getFileList() ([]string, int) {

	files := []string{}
	length := 0

	// Find NSP files within the scanNodes list of directories
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Failed to process %s: %s", path, err)
			return nil
		}
		if info.IsDir() {
			return nil
		}

		switch filepath.Ext(info.Name()) {
		case ".nsp", ".nsz", ".xci":
			relPath := strings.TrimPrefix(path, rootPath+"/")
			fileURL := fmt.Sprintf("%s:%s/files/%s\n", externalHost, externalPort, url.PathEscape(relPath))
			files = append(files, fileURL)
			length += len(fileURL)
		default:
			log.Printf("Ignoring %s - unknown file extension.", path)
		}
		return nil
	})
	if err != nil {
		log.Printf("could not scan folders: %v", err)
		return nil, 0
	}

	log.Printf("Found %v files total.", len(files))
	return files, length
}

func sendFileList(out io.Writer) {
	fileList, length := getFileList()

	log.Printf("Sending file list...")

	// Taken from : https://github.com/bycEEE/tinfoilusbgo/blob/master/main.go. Adapted to work for network (bigEndian)
	// sendNSPList creates a payload out of an NSPList struct and sends it to the switch.
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(length)) // NSP list length
	out.Write(buf)

	for _, path := range fileList {
		buf = make([]byte, len(path))
		copy(buf, path) // File path followed by newline
		out.Write(buf)
	}

	log.Printf("File list sent.")
}
