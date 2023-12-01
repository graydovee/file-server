package pkg

import (
	"fmt"
	"github.com/google/uuid"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FileServer struct {
	serveMux *http.ServeMux

	address string

	uploadDir string

	enableTls bool
}

func NewFileServer(address string, uploadDir string, enableTls bool) (*FileServer, error) {
	absUploadDir, err := filepath.Abs(uploadDir)
	if err != nil {
		return nil, err
	}
	return &FileServer{
		address:   address,
		uploadDir: absUploadDir,
		enableTls: enableTls,
	}, nil
}

func (f *FileServer) Run() error {

	f.serveMux = http.NewServeMux()
	f.serveMux.HandleFunc("/", f.handleHelpPage)
	f.serveMux.HandleFunc("/upload", f.uploadFileHandler)
	f.serveMux.HandleFunc("/download/", f.downloadFileHandler)

	log.Printf("Server started at %s\n", f.address)
	log.Printf("Upload directory: %s\n", f.uploadDir)
	if err := http.ListenAndServe(f.address, f.serveMux); err != nil {
		return err
	}
	return nil
}

func (f *FileServer) uploadFileHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("File uploaded")
	if r.Method != http.MethodPost {
		log.Printf("Only POST method is allowed: %s\n", r.Method)
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse uploaded file
	file, header, err := r.FormFile("file")
	if err != nil {
		log.Printf("Error retrieving the file: %s\n", err.Error())
		http.Error(w, "Error retrieving the file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Generate UUID for filename and remove dashes
	newFileName := strings.ReplaceAll(uuid.New().String(), "-", "") + filepath.Ext(header.Filename)

	// Create directory structure based on current year and month
	now := time.Now()
	yearMonthPath := fmt.Sprintf("%d/%02d", now.Year(), now.Month())
	yearMonthPathDir := filepath.Join(f.uploadDir, yearMonthPath)
	if err := os.MkdirAll(yearMonthPathDir, os.ModePerm); err != nil {
		log.Printf("Error creating directory %s: %s\n", yearMonthPathDir, err.Error())
		http.Error(w, "Error creating directory", http.StatusInternalServerError)
		return
	}

	filePath := filepath.Join(yearMonthPath, newFileName)

	fullPath := filepath.Join(f.uploadDir, filePath)

	// Create new file
	newFile, err := os.Create(fullPath)
	if err != nil {
		log.Printf("Error creating the file %s: %s\n", filePath, err.Error())
		http.Error(w, "Error creating the file", http.StatusInternalServerError)
		return
	}
	defer newFile.Close()

	// Copy the uploaded file to the new file
	_, err = io.Copy(newFile, file)
	if err != nil {
		log.Printf("Error saving the file %s: %s\n", filePath, err.Error())
		http.Error(w, "Error saving the file", http.StatusInternalServerError)
		return
	}

	log.Printf("File %s uploaded successfully: %s\n", header.Filename, filePath)
	_, _ = fmt.Fprintf(w, "File uploaded successfully.\n Download command: wget %s -O %s\n", getDownloadAddress(r.Host, filePath, f.enableTls), header.Filename)
}

func (f *FileServer) downloadFileHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the file name from the URL path
	fileName := r.URL.Path
	if fileName == "" {
		http.Error(w, "File name is required", http.StatusBadRequest)
		return
	}

	fullFilePath := filepath.Join(f.uploadDir, strings.TrimPrefix(fileName, "/download"))
	log.Printf("Download file: %s\n", fileName)

	http.ServeFile(w, r, fullFilePath)
}

type PageData struct {
	UploadAddress   string
	DownloadAddress string
}

func NewPageData(host string, enableTls bool) *PageData {
	return &PageData{
		UploadAddress:   getUploadAddress(host, enableTls),
		DownloadAddress: getDownloadAddress(host, "[year]/[month]/[file_name]", enableTls),
	}
}

func getUploadAddress(host string, enableTls bool) string {
	var schema string
	if enableTls {
		schema = "https"
	} else {
		schema = "http"
	}
	return fmt.Sprintf("%s://%s/upload", schema, host)
}

func getDownloadAddress(host, filePath string, enableTls bool) string {
	var schema string
	if enableTls {
		schema = "https"
	} else {
		schema = "http"
	}
	return fmt.Sprintf("%s://%s/download/%s", schema, host, filePath)
}

func (f *FileServer) handleHelpPage(w http.ResponseWriter, r *http.Request) {
	data := NewPageData(r.Host, f.enableTls)
	// 解析模板文件
	tmpl, err := template.ParseFiles("assert/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 渲染模板并发送响应
	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
