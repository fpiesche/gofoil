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
var switchHostsArg string
var switchHosts []string
var listenAddress string
var externalAddress string
var pollInterval int

func TinfoilExtensions() []string {
	return []string{".nsp", ".nsz", ".xci"}
}

func FBIExtensions() []string {
	return []string{".cia", ".tik"}
}

func readArgs() {
	flagset := flag.NewFlagSetWithEnvPrefix(os.Args[0], "GOFOIL", 0)

	flagset.StringVar(&rootPath, "root", "/games", "Root path for files to serve")
	flagset.StringVar(&switchHostsArg, "switchhosts", "localhost", "IP addresses or host names for Switches to check.")
	flagset.StringVar(&listenAddress, "listenaddress", "0.0.0.0:8000", "IP address to bind server to.")
	flagset.StringVar(&externalAddress, "externaladdress", "0.0.0.0:8000", "External IP address or host name to create download links on.")
	flagset.IntVar(&pollInterval, "pollinterval", 2, "How often to poll for the presence of Switches.")
	flagset.Parse(os.Args[1:])
	switchHosts = strings.Split(switchHostsArg, ",")
}


func main() {

	readArgs()
	r := mux.NewRouter()
	r.HandleFunc("/", HealthcheckHandler)
	r.PathPrefix("/files/").Handler(http.StripPrefix("/files/", http.FileServer(http.Dir(rootPath))))

	log.Printf("Starting server at %s, external address %s.", listenAddress, externalAddress)
	log.Printf("Games root path: %s", rootPath)
	log.Printf("Switch hosts to poll: %s", switchHostsArg)

	srv := &http.Server{
		Handler: r,
		Addr:    listenAddress,
		// Good practice: enforce timeouts for servers you create!
		// But here the switch will be downloading file, slowly...
		//WriteTimeout: 15 * time.Second,
		//ReadTimeout:  15 * time.Second,
	}

	for _, switchHost := range switchHosts {
		go pollForTinfoil(switchHost)
	}

	err := srv.ListenAndServe()
	if err != nil {
		log.Panic(err)
	}
}

func HealthcheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusCreated)
}

func pollForTinfoil(switchHost string) {
	poll := time.Tick(time.Second * time.Duration(pollInterval))
	for _ = range poll {
		conn, err := net.DialTimeout("tcp", switchHost + ":2000", time.Second * time.Duration(pollInterval - 1))
		if err == nil {
			log.Printf("%s: found Tinfoil at %s", switchHost, conn.RemoteAddr())
			defer conn.Close()
			files, length := getFileList(rootPath, TinfoilExtensions())
			sendFileList(conn, files, length)
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
