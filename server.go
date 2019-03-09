package main

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

type appHandler struct {
	title   string
	content string
}

type pageVariables struct {
	Content   template.HTML
}

// For random strings (filenames)
var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// Lock for uploadHandler
var mutex sync.Mutex

// Serve HTTP based on passed appHandler struct with content
func (templateinfo *appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	content := template.HTML(templateinfo.content)

	// Create a new struct of data to pass into template.html
	TemplateVars := pageVariables{
		Content:   content,
	}

	// Parse HTML template
	t, err := template.ParseFiles("template.html")
	if err != nil {
		log.Print("Template parsing error: ", err)
	}

	// Write to template -- this will be served as HTML to the user
	err = t.Execute(w, TemplateVars)
	if err != nil {
		log.Print("Template executing error: ", err)
	}
}

// Handle user uploads
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()
	fmt.Println("method:", r.Method)
	if r.Method == "GET" {
		http.Redirect(w, r, "", 301)
		return

	} else {
		// Accept input from HTML file upload form. Used with
		// upload.html.
		// File size limits
		r.ParseMultipartForm(32 << 20)
		uploadedFile, handler, err := r.FormFile("uploadfile")
		if err != nil {
			log.Println(err)
			return
		}

		// Write file to disk
		// Use a randomly-generated meaningless filename
		defer uploadedFile.Close()
		random := randSeq(30)

		// Check if file exists
		for {
			if _, err := os.Stat("./uploads/" + random); err == nil {
				// Generate new name if exists
				random = randSeq(30)
				continue

			} else if os.IsNotExist(err) {
				// File does not exist, safe to proceed
				f, err := os.OpenFile("./uploads/"+random,
					os.O_WRONLY|os.O_CREATE, 0666)
				if err != nil {
					log.Println(err)
					return
				}

				// Write to server's disk
				defer f.Close()
				io.Copy(f, uploadedFile)
				break

			} else {
				// Schrodinger's file... Doesn't exist and doesn't not exist
				log.Println(err)
				return
			}

		}

		// Post-upload file processing
		uploadType := r.PostFormValue("uploadtype") // From upload.html

		if uploadType == "references" {

			refname := r.PostFormValue("referencesname") // From upload.html
			afterRefname := r.PostFormValue("afterref")
			cmd := exec.Command("./docx_refs.py", "./uploads/"+random, refname, afterRefname)
			out, err := cmd.Output()
			if err != nil {
				log.Printf("cmd.Run() failed with %s\n", err)
				return
			}
			outStr := string(out)

			// Return error message (stdout of docx.py) if applicable
			if len(outStr) > 0 {
				log.Print(outStr)
				title := "References checker"
				titlelink := strings.ToLower(title)
				titlelink = strings.Replace(titlelink, " ", "-", -1)

				// References
				uploadHTMLBytes, readErr := ioutil.ReadFile("upload.html")
				if readErr != nil {
					fmt.Print(readErr)
				}

				uploadHTML := string(uploadHTMLBytes)
				// Red color for error message contained in outStr
				uploadHTML = uploadHTML + "<p><font color = #bb0000>" + outStr + "</font></p>"
				content := template.HTML(uploadHTML)

				// Struct for template.html
				TemplateVars := pageVariables{
					Content:   content,
				}

				// Parse HTML template
				t, err := template.ParseFiles("template.html")
				if err != nil {
					log.Print("Template parsing error: ", err)
				}

				// Write to template -- this will be served as HTML to the user
				err = t.Execute(w, TemplateVars)
				if err != nil {
					log.Print("Template executing error: ", err)
				}
				return
			}
		} else {
			// Invalid request, redirect to homepage
			http.Redirect(w, r, "", 301)
			return
		}

		// Return processed file to client
		// First, check if file exists and can be opened
		clientFile, err := os.Open("./uploads/" + random)
		defer clientFile.Close()
		if err != nil {
			http.Error(w, "File not found.", 404)
			return
		}

		// File is found, create and send the correct headers
		// Get the Content-Type of the file
		// Create a buffer to store the header of the file in
		fileHeader := make([]byte, 512)

		// Copy the headers into the fileHeader buffer
		clientFile.Read(fileHeader)

		// Get content type of file
		fileContentType := http.DetectContentType(fileHeader)

		// Get the file size
		fileStat, _ := clientFile.Stat()
		fileSize := strconv.FormatInt(fileStat.Size(), 10)

		// Send the headers
		fmt.Printf(handler.Filename) // temp
		w.Header().Set("Content-Disposition", "attachment; filename=response.html")
		w.Header().Set("Content-Type", fileContentType)
		w.Header().Set("Content-Length", fileSize)

		// Send the file
		clientFile.Seek(0, 0)
		io.Copy(w, clientFile) // Send file to the client with http.ResponseWriter
		return
	}
}

func main() {
	mux := http.NewServeMux()

	// Reference-checker page
	uploadHTMLBytes, readErr := ioutil.ReadFile("upload.html")
	if readErr != nil {
		fmt.Print(readErr)
	}

	uploadHTML := string(uploadHTMLBytes)
	ref := &appHandler{content: uploadHTML}
	mux.Handle("/", ref)

	// Handle uploads
	mux.HandleFunc("/upload", uploadHandler)

	// Log
	log.Println("Listening...")

	err := http.ListenAndServe("localhost:8000", mux)
	
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
