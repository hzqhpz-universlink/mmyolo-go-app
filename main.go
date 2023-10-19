package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type PageData struct {
    ImageURL string
    TotalFaces int
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
    tmpl := template.Must(template.ParseFiles("index.html"))
    data := PageData{}
    tmpl.Execute(w, data)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20) // 10 MB limit

	file, handler, err := r.FormFile("file")
	if err != nil {
		fmt.Println("Error retrieving the file:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

    // Generate a unique filename to prevent overwriting
    filename := strconv.FormatInt(time.Now().UnixNano(), 10) + "_" + handler.Filename

    // Create a new file in the server's uploads directory
    newFile, err := os.Create("data/input/" + filename)
    if err != nil {
        fmt.Println("Error creating the file:", err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer newFile.Close()

    // Copy the uploaded file to the new file on the server
    _, err = io.Copy(newFile, file)
    if err != nil {
        fmt.Println("Error copying the file:", err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    err = runYoloDetection(filename)

    if err != nil {
        fmt.Println("Error running face detection:", err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    imageURL := "/results/" + filename

	// // Read the JSON data from the file
	// total_data, err := ioutil.ReadFile("total_faces.json")
	// if err != nil {
	// 	fmt.Println("Error reading JSON file:", err)
	// 	return
	// }

	// // Parse the JSON data into a map
	// var faceData map[string]int
	// if err := json.Unmarshal(total_data, &faceData); err != nil {
	// 	fmt.Println("Error parsing JSON:", err)
	// 	return
	// }

	// // Access data based on the file name
	// totalFaces, exists := faceData[filename]
	// if exists {
	// 	fmt.Printf("File '%s' has %d faces.\n", filename, totalFaces)
	// } else {
	// 	fmt.Printf("File '%s' not found in the data.\n", filename)
	// }

    // data := PageData{ImageURL: imageURL, TotalFaces: totalFaces}
	data := PageData{ImageURL: imageURL}
    
    tmpl := template.Must(template.ParseFiles("index.html"))
    tmpl.Execute(w, data)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func runYoloDetection(filename string) error {

	// Define the name of the Docker container
	containerName := "mmyolo-go"

    // Define the commands to execute
    mimDownloadCommand := "mim download mmyolo --config yolov5_s-v61_syncbn_fast_8xb16-300e_coco --dest ."
    pythonDemoCommand := "python demo/image_demo.py data/input/" + filename + " yolov5_s-v61_syncbn_fast_8xb16-300e_coco.py yolov5_s-v61_syncbn_fast_8xb16-300e_coco_20220918_084700-86e02187.pth --device cpu --out-dir ./data/output"

    // Execute the commands inside the Docker container
    cmd := exec.Command("docker", "exec", "-i", containerName, "sh", "-c", mimDownloadCommand + " && " + pythonDemoCommand)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    if err := cmd.Run(); err != nil {
        log.Fatalf("Error running the commands: %v", err)
    }

	return nil
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/", indexHandler).Methods("GET")
	r.HandleFunc("/upload", uploadHandler).Methods("POST")

	// Create an "uploads" directory for file storage
	os.Mkdir("uploads", 0755)

	http.Handle("/", r)
	http.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("data/input"))))
    http.Handle("/results/", http.StripPrefix("/results/", http.FileServer(http.Dir("data/output"))))

    log.Println("Starting server on port 8080...")

	http.ListenAndServe(":8080", nil)
}