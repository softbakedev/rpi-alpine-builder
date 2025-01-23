package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"softbake.dev/rpialp/internal"
)

const (
	AppName             = "rpialp"
	VolumeLabel  string = "ALPINE"
	VolumeSizeMg int    = 4096
)

type RpiAlpineParams struct {
	Version         string `json:"version"`
	Hostname        string `json:"hostname"`
	WifiNetwork     string `json:"wifiNetwork"`
	WifiPassword    string `json:"wifiPassword"`
	RootPassword    string `json:"rootPassword"`
	VolumeDirectory string `json:"volumeDirectory"`
}

// App struct
type App struct {
	ctx    context.Context
	server *http.Server
}

func setupDefaultHeaders(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*") // Replace '*' with specific origin in production
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Content-Type", "application/json")
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// Startup is called at application startup
func (a *App) Startup(ctx context.Context) {
	// Perform your setup here
	a.ctx = ctx

	// Initialize HTTP server and routes
	mux := http.NewServeMux()
	mux.HandleFunc("/api/alpine-versions", a.handleAlpineVersions)
	mux.HandleFunc("/api/volume-directories", a.handleVolumeDirectories)
	//mux.HandleFunc("/api/wifi-networks", a.handleWifiNetworks)
	mux.HandleFunc("/api/build", a.handleBuild)

	a.server = &http.Server{
		Addr:    ":8080", // You can change the port if needed
		Handler: mux,
	}

	// Start the server in a new goroutine
	go func() {
		log.Printf("Starting server on %s", a.server.Addr)
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Could not listen on %s: %v\n", a.server.Addr, err)
		}
	}()
}

// Shutdown is called at application termination
func (a *App) Shutdown(ctx context.Context) {
	// Perform your teardown here
	if a.server != nil {
		log.Println("Shutting down server...")
		// Attempt a graceful shutdown
		if err := a.server.Shutdown(ctx); err != nil {
			log.Fatalf("Server Shutdown Failed:%+v", err)
		}
		log.Println("Server gracefully stopped")
	}
}

// DomReady is called after front-end resources have been loaded
func (a *App) DomReady(ctx context.Context) {
	// Add your action here
}

// BeforeClose is called when the application is about to quit,
// either by clicking the window close button or calling runtime.Quit.
// Returning true will cause the application to continue, false will continue shutdown as normal.
func (a *App) BeforeClose(ctx context.Context) (prevent bool) {
	return false
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// Handler for /api/alpine-versions
func (a *App) handleAlpineVersions(w http.ResponseWriter, r *http.Request) {

	setupDefaultHeaders(w, r)

	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	versions, err := internal.GetAlpineVersions()

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error get alpine versions")
	}

	if err := json.NewEncoder(w).Encode(versions); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal Server Error")
	}
}

// Handler for /api/volume-directories
func (a *App) handleVolumeDirectories(w http.ResponseWriter, r *http.Request) {

	setupDefaultHeaders(w, r)

	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	volumes, err := internal.ListBlockDevices()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error get volumes/directories")
	}

	if err := json.NewEncoder(w).Encode(volumes); err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error decode json volumes"))
	}
}

func validateRpiAlpineParams(rpiAlpineParams RpiAlpineParams) error {
	if len(rpiAlpineParams.Version) == 0 {
		return errors.New("attribute version cannot be empty")
	}
	if len(rpiAlpineParams.Hostname) == 0 {
		return errors.New("attribute hostname cannot be empty")
	}
	if len(rpiAlpineParams.WifiNetwork) == 0 {
		return errors.New("attribute wifi name cannot be empty")
	}

	if len(rpiAlpineParams.WifiPassword) == 0 {
		return errors.New("attribute wifi password cannot be empty")
	}

	if len(rpiAlpineParams.RootPassword) == 0 {
		return errors.New("attribute root password cannot be empty")
	}

	if len(rpiAlpineParams.VolumeDirectory) == 0 {
		return errors.New("attribute volume cannot be empty")
	}
	return nil
}

// Handler for /api/build
func (a *App) handleBuild(w http.ResponseWriter, r *http.Request) {
	log.Println("Calling /api/build")

	setupDefaultHeaders(w, r)

	// Handle OPTIONS requests for CORS preflight
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Allow only POST requests
	if r.Method != http.MethodPost {
		log.Printf("Method %v", r.Method)
		respondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	// Initialize the struct to hold parsed JSON data
	var rpiAlpineParams RpiAlpineParams

	// Decode JSON payload
	if err := json.NewDecoder(r.Body).Decode(&rpiAlpineParams); err != nil {
		log.Printf("Error decoding JSON: %v", err)
		http.Error(w, fmt.Sprintf("Error decoding build params: %v", err), http.StatusBadRequest)
		return
	}

	// Validate the decoded JSON data
	if err := validateRpiAlpineParams(rpiAlpineParams); err != nil {
		log.Printf("Validation error: %v", err)
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid input: %v", err))
		return
	}

	// Example: Proceed with your application logic
	if err := internal.DownloadAlpineRelease(AppName, rpiAlpineParams.Version); err != nil {
		log.Printf("Error: %v", err)
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed downloading Alpine release: %v", err))
		return
	}

	ssidPsk, err := internal.GenerateWpaPsk(rpiAlpineParams.WifiNetwork, rpiAlpineParams.WifiPassword)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed generating WPA2-PSK: %v", err))
		return
	}

	shadowPass, err := internal.GenerateShadowPasswordHash(rpiAlpineParams.RootPassword)
	if err != nil {
		log.Printf("Error encrypting root password: %v", err)
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to encrypt root password: %v", err))
		return
	}

	if err := internal.ProcessApkovl(AppName, rpiAlpineParams.Hostname, rpiAlpineParams.WifiNetwork, ssidPsk, shadowPass); err != nil {
		log.Printf("Error processing APKOVL: %v", err)
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to process APKOVL tar file: %v", err))
		return
	}

	if err = internal.FormatVolumeFat32(rpiAlpineParams.VolumeDirectory, VolumeLabel, VolumeSizeMg); err != nil {
		log.Printf("Error formatting FAT32 volume: %v", err)
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error formatting FAT32 volume %s: %v", rpiAlpineParams.VolumeDirectory, err))
		return
	}

	if err = internal.BuildImage(AppName, rpiAlpineParams.VolumeDirectory, rpiAlpineParams.Hostname); err != nil {
		log.Printf("Error building image: %v", err)
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error building image in volume %s: %v", rpiAlpineParams.VolumeDirectory, err))
		return
	}

	// Success response
	respondWithSuccess(w, "Build completed successfully")
}

//// Handler for /api/wifi-networks
//func (a *App) handleWifiNetworks(w http.ResponseWriter, r *http.Request) {
//	if r.Method != http.MethodGet {
//		respondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
//		return
//	}
//
//	setupDefaultHeaders(w, r)
//
//	wifiNetworks := []string{"Network1", "Network2", "Network3"}
//
//	if err := json.NewEncoder(w).Encode(wifiNetworks); err != nil {
//		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
//	}
//}

// ErrorResponse defines the structure for error responses
type ErrorResponse struct {
	Error string `json:"error"`
}

// SuccessResponse defines the structure for successful responses
type SuccessResponse struct {
	Message string `json:"message"`
}

// respondWithError sends a JSON-formatted error response
func respondWithError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResp := ErrorResponse{
		Error: message,
	}

	if err := json.NewEncoder(w).Encode(errorResp); err != nil {
		log.Printf("Failed to write error response: %v", err)
	}
}

// respondWithSuccess sends a JSON-formatted success response
func respondWithSuccess(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	successResp := SuccessResponse{
		Message: message,
	}

	if err := json.NewEncoder(w).Encode(successResp); err != nil {
		log.Printf("Failed to write success response: %v", err)
	}
}
