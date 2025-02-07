package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type CrewForm struct {
	Nama                 string   `json:"nama"`
	Nik                  string   `json:"nik"`
	JenisKelamin         string   `json:"jenis_kelamin"`
	Domisili             []string `json:"domisili"`
	Usia                 string   `json:"usia"`
	NomorHp              string   `json:"nomor_hp"`
	Email                string   `json:"email"`
	Agama                string   `json:"agama"`
	StatusNikah          string   `json:"status_nikah"`
	SeafarerCode         string   `json:"seafarer_code"`
	NomorPassport        string   `json:"nomor_passport"`
	Ijazah               string   `json:"ijazah"`
	TahunPengalamanKerja string   `json:"tahun_pengalaman_kerja"`
	JabatanTerakhir      string   `json:"jabatan_terakhir"`
	JenisKapal           string   `json:"jenis_kapal"`
	PengalamanDiMigas    string   `json:"pengalaman_di_migas"`
	UpahSaatIni          string   `json:"upah_saat_ini"`
	EkspektasiUpah       string   `json:"ekspektasi_upah"`
	LampiranCv           string   `json:"lampiran_cv"`
	LampiranFoto         string   `json:"lampiran_foto"`
	Sertifikat           []string `json:"sertifikat"`
}

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

var (
	crewForms     = make([]CrewForm, 0)
	mc            sync.Mutex
	driveService  *drive.Service
	sheetsService *sheets.Service
	spreadsheetId string
)

func initEnv() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	spreadsheetId = os.Getenv("SPREADSHEET_ID")
}

// initGoogleDrive initializes the Google Drive service
func initGoogleDrive() error {
	ctx := context.Background()

	credBytes, err := os.ReadFile("credentials.json")
	if err != nil {
		return fmt.Errorf("unable to read credentials file: %v", err)
	}

	config, err := google.JWTConfigFromJSON(credBytes, drive.DriveFileScope)
	if err != nil {
		return fmt.Errorf("unable to parse credentials: %v", err)
	}

	service, err := drive.NewService(ctx, option.WithHTTPClient(config.Client(ctx)))
	if err != nil {
		return fmt.Errorf("unable to create drive service: %v", err)
	}

	driveService = service
	return nil
}

// uploadFileToDrive uploads a base64 encoded file to Google Drive
func uploadFileToDrive(base64Data, fileName, mimeType string) (string, error) {
	// Remove data URI prefix
	base64Data = strings.Split(base64Data, ",")[1]

	// Decode base64 data
	fileData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 data: %v", err)
	}

	file := &drive.File{
		Name:     fileName,
		MimeType: mimeType,
	}

	// Create the file in Google Drive
	f, err := driveService.Files.Create(file).
		Media(strings.NewReader(string(fileData))).
		Do()
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %v", err)
	}

	// Create a shareable link
	permission := &drive.Permission{
		Type: "anyone",
		Role: "reader",
	}

	_, err = driveService.Permissions.Create(f.Id, permission).Do()
	if err != nil {
		return "", fmt.Errorf("failed to set file permissions: %v", err)
	}

	return fmt.Sprintf("https://drive.google.com/file/d/%s/view", f.Id), nil
}

// initGoogleSheets initializes the Google Sheets service
func initGoogleSheets() error {
	ctx := context.Background()

	credBytes, err := os.ReadFile("credentials.json")
	if err != nil {
		return fmt.Errorf("unable to read credentials file: %v", err)
	}

	config, err := google.JWTConfigFromJSON(credBytes, sheets.SpreadsheetsScope)
	if err != nil {
		return fmt.Errorf("unable to parse credentials: %v", err)
	}

	service, err := sheets.NewService(ctx, option.WithHTTPClient(config.Client(ctx)))
	if err != nil {
		return fmt.Errorf("unable to create sheets service: %v", err)
	}

	sheetsService = service

	// Initialize headers if needed
	initializeHeaders()
	return nil
}

// initializeHeaders creates headers in the spreadsheet if they don't exist
func initializeHeaders() {
	headers := &sheets.ValueRange{
		Values: [][]interface{}{{
			"Nama", "NIK", "Jenis Kelamin", "Domisili", "Usia", "Nomor HP", "Email", "Agama", "Status Nikah", "Seafarer Code", "Nomor Passport", "Ijazah", "Tahun Pengalaman Kerja", "Jabatan Terakhir", "Jenis Kapal", "Pengalaman di Migas", "Upah Saat Ini", "Ekspektasi Upah", "Lampiran CV", "Lampiran Foto", "Sertifikat", "Timestamp",
		}},
	}

	sheetsService.Spreadsheets.Values.Update(
		spreadsheetId,
		"A1:V1",
		headers,
	).ValueInputOption("RAW").Do()
}

// writeToSheet writes the form data to Google Sheets
func writeToSheet(crew CrewForm) error {
	values := []interface{}{
		crew.Nama,
		crew.Nik,
		crew.JenisKelamin,
		strings.Join(crew.Domisili, ", "),
		crew.Usia,
		crew.NomorHp,
		crew.Email,
		crew.Agama,
		crew.StatusNikah,
		crew.SeafarerCode,
		crew.NomorPassport,
		crew.Ijazah,
		crew.TahunPengalamanKerja,
		crew.JabatanTerakhir,
		crew.JenisKapal,
		crew.PengalamanDiMigas,
		crew.UpahSaatIni,
		crew.EkspektasiUpah,
		crew.LampiranCv,
		crew.LampiranFoto,
		strings.Join(crew.Sertifikat, ", "),
		fmt.Sprintf("=NOW()"), // Adds timestamp
	}

	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{values},
	}

	_, err := sheetsService.Spreadsheets.Values.Append(
		spreadsheetId,
		"A1", // Starting cell reference
		valueRange,
	).ValueInputOption("USER_ENTERED").InsertDataOption("INSERT_ROWS").Do()

	return err
}

func main() {

	// Initialize environment variables
	initEnv()

	// Use ServeMux
	mux := http.NewServeMux()

	// API Routes
	mux.HandleFunc("/api/provinces", handleProvinceNames)
	mux.HandleFunc("/api/cities", handleCityNames)

	// Web Routes
	mux.HandleFunc("/", handleHome)
	mux.HandleFunc("/submit", func(w http.ResponseWriter, r *http.Request) {
		r.ParseMultipartForm(10 << 20) // 10 MB max memory
		handleSubmit(w, r)
	})

	// Initialize Google Sheets API
	if err := initGoogleSheets(); err != nil {
		log.Printf("Warning: Google Sheets integration failed: %v", err)
		log.Println("Continuing without Google Sheets integration...")
	}

	// Initialize Google Drive API
	if err := initGoogleDrive(); err != nil {
		log.Printf("Warning: Google Drive integration failed: %v", err)
		log.Println("Continuing without Google Drive integration...")
	}

	http.HandleFunc("/", handleHome)
	http.HandleFunc("/submit", func(w http.ResponseWriter, r *http.Request) {
		r.ParseMultipartForm(10 << 20) // 10 MB max memory
		handleSubmit(w, r)
	})

	fmt.Println("Server started at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/index.html"))
	tmpl.Execute(w, nil)
}

func handleSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse form data
	crewForm := CrewForm{
		Nama:                 r.FormValue("nama"),
		Nik:                  r.FormValue("nik"),
		JenisKelamin:         r.FormValue("jenis_kelamin"),
		Domisili:             r.Form["domisili[]"],
		Usia:                 r.FormValue("usia"),
		NomorHp:              r.FormValue("nomor_hp"),
		Email:                r.FormValue("email"),
		Agama:                r.FormValue("agama"),
		StatusNikah:          r.FormValue("status_nikah"),
		SeafarerCode:         r.FormValue("seafarer_code"),
		NomorPassport:        r.FormValue("nomor_passport"),
		Ijazah:               r.FormValue("ijazah"),
		TahunPengalamanKerja: r.FormValue("tahun_pengalaman_kerja"),
		JabatanTerakhir:      r.FormValue("jabatan_terakhir"),
		JenisKapal:           r.FormValue("jenis_kapal"),
		PengalamanDiMigas:    r.FormValue("pengalaman_di_migas"),
		UpahSaatIni:          r.FormValue("upah_saat_ini"),
		EkspektasiUpah:       r.FormValue("ekspektasi_upah"),
		LampiranCv:           r.FormValue("lampiran_cv"),
		LampiranFoto:         r.FormValue("lampiran_foto"),
		Sertifikat:           r.Form["sertifikat[]"],
	}

	// Process CV upload
	cvLink := ""
	if crewForm.LampiranCv != "" {
		cvMimeType := ""
		if strings.Contains(crewForm.LampiranCv, "pdf") {
			cvMimeType = "application/pdf"
		} else {
			cvMimeType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
		}

		fileName := fmt.Sprintf("CV_%s_%s%s",
			crewForm.Nama,
			crewForm.Nik,
			getFileExtension(cvMimeType))

		uploadedLink, err := uploadFileToDrive(crewForm.LampiranCv, fileName, cvMimeType)
		if err != nil {
			response := Response{
				Success: false,
				Message: fmt.Sprintf("Failed to upload CV: %v", err),
			}
			json.NewEncoder(w).Encode(response)
			return
		}
		cvLink = uploadedLink
	}

	// Process photo upload
	photoLink := ""
	if crewForm.LampiranFoto != "" {
		photoMimeType := ""
		if strings.Contains(crewForm.LampiranFoto, "jpeg") || strings.Contains(crewForm.LampiranFoto, "jpg") {
			photoMimeType = "image/jpeg"
		} else {
			photoMimeType = "image/png"
		}

		fileName := fmt.Sprintf("Photo_%s_%s%s",
			crewForm.Nama,
			crewForm.Nik,
			getFileExtension(photoMimeType))

		uploadedLink, err := uploadFileToDrive(crewForm.LampiranFoto, fileName, photoMimeType)
		if err != nil {
			response := Response{
				Success: false,
				Message: fmt.Sprintf("Failed to upload photo: %v", err),
			}
			json.NewEncoder(w).Encode(response)
			return
		}
		photoLink = uploadedLink
	}

	// Update form with Google Drive links
	crewForm.LampiranCv = cvLink
	crewForm.LampiranFoto = photoLink

	if err := validateCrewForm(crewForm); err != nil {
		response := Response{
			Success: false,
			Message: err.Error(),
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Store in local memory
	mc.Lock()
	crewForms = append(crewForms, crewForm)
	mc.Unlock()

	// Write to Google Sheets if service is available
	if sheetsService != nil {
		if err := writeToSheet(crewForm); err != nil {
			log.Printf("Failed to write to Google Sheets: %v", err)
			// Continue execution - don't return error to user if only Google Sheets fails
		}
	}

	response := Response{
		Success: true,
		Message: "Crew data has been submitted successfully!",
	}
	json.NewEncoder(w).Encode(response)
}

func getFileExtension(mimeType string) string {
	extensions, err := mime.ExtensionsByType(mimeType)
	if err != nil || len(extensions) == 0 {
		switch mimeType {
		case "application/pdf":
			return ".pdf"
		case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
			return ".docx"
		case "image/jpeg":
			return ".jpg"
		case "image/png":
			return ".png"
		default:
			return ""
		}
	}
	return extensions[0]
}

var client = &http.Client{
	Timeout: 10 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return nil
	},
}

func fetchWithRedirect(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; GoClient/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func handleProvinceNames(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	data, err := fetchWithRedirect("https://www.emsifa.com/api-wilayah-indonesia/api/provinces.json")

	if err != nil {
		fmt.Println("Error fetching provinces:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(data)
}

func handleCityNames(w http.ResponseWriter, r *http.Request) {
	provinceID := r.URL.Query().Get("provinceId")
	if provinceID == "" {
		http.Error(w, "Province ID is required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	url := fmt.Sprintf("https://www.emsifa.com/api-wilayah-indonesia/api/regencies/%s.json", provinceID)
	data, err := fetchWithRedirect(url)
	if err != nil {
		fmt.Println("Error fetching regencies:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(data)
}

func validateCrewForm(crewForm CrewForm) error {
	// Validasi nama, harus ada isinya
	if crewForm.Nama == "" {
		return fmt.Errorf("name is required")
	}

	// Validasi NIK, harus angka 16 digit
	if matched, _ := regexp.MatchString(`^\d{16}$`, crewForm.Nik); !matched {
		return fmt.Errorf("NIK must be 16 digits")
	}

	// Validasi jenis kelamin
	if crewForm.JenisKelamin == "" {
		return fmt.Errorf("gender is required")
	}

	// Validasi domisili
	if len(crewForm.Domisili) == 0 {
		return fmt.Errorf("domicile is required")
	}

	// Validasi usia
	ageRegex := regexp.MustCompile(`^\d{1,2}$`)
	if crewForm.Usia == "" {
		return fmt.Errorf("age is required")
	}
	if !ageRegex.MatchString(crewForm.Usia) {
		return fmt.Errorf("age must be numeric and between 1-99")
	}

	// Validasi Nomor HP, diantara 10 - 13 digit
	if crewForm.NomorHp == "" {
		return fmt.Errorf("phone number is required")
	}

	// Validasi email, harus ada @
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(crewForm.Email) {
		return fmt.Errorf("email is required")
	}

	// Validasi agama
	if crewForm.Agama == "" {
		return fmt.Errorf("religion is required")
	}

	// Validasi status nikah
	if crewForm.StatusNikah == "" {
		return fmt.Errorf("marital status is required")
	}

	// Validasi seafarer code
	if crewForm.SeafarerCode == "" {
		return fmt.Errorf("seafarer code is required")
	}

	// Validasi nomor passport
	if crewForm.NomorPassport == "" {
		return fmt.Errorf("passport number is required")
	}

	// Validasi ijazah
	if crewForm.Ijazah == "" {
		return fmt.Errorf("diploma is required")
	}

	// Validasi tahun pengalaman kerja
	if crewForm.TahunPengalamanKerja == "" {
		return fmt.Errorf("years of work experience is required")
	}

	// Validasi jabatan terakhir
	if crewForm.JabatanTerakhir == "" {
		return fmt.Errorf("last position is required")
	}

	// Validasi jenis kapal
	if crewForm.JenisKapal == "" {
		return fmt.Errorf("ship type is required")
	}

	// Validasi pengalaman di migas
	if crewForm.PengalamanDiMigas == "" {
		return fmt.Errorf("experience in oil and gas is required")
	}

	// Validasi upah saat ini
	if crewForm.UpahSaatIni == "" {
		return fmt.Errorf("current wage is required")
	}

	// Validasi ekspektasi upah
	if crewForm.EkspektasiUpah == "" {
		return fmt.Errorf("wage expectation is required")
	}

	// Validasi lampiran CV
	if crewForm.LampiranCv == "" {
		return fmt.Errorf("CV attachment is required")
	}

	// Validasi lampiran foto
	if crewForm.LampiranFoto == "" {
		return fmt.Errorf("photo attachment is required")
	}

	// Validasi sertifikat
	if len(crewForm.Sertifikat) == 0 {
		return fmt.Errorf("at least one certificate is required")
	}

	return nil
}
