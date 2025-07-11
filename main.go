package main // Define the main package

import (
	"bytes"         // Provides bytes support
	"io"            // Provides basic interfaces to I/O primitives
	"log"           // Provides logging functions
	"net/http"      // Provides HTTP client and server implementations
	"net/url"       // Provides URL parsing and encoding
	"os"            // Provides functions to interact with the OS (files, etc.)
	"path/filepath" // Provides filepath manipulation functions
	"regexp"        // Provides regex support functions.
	"strings"       // Provides string manipulation functions
	"time"          // Provides time-related functions

	"golang.org/x/net/html"
)

func main() {
	// Remote API URL.
	remoteAPIURL := "https://en.walkera.com/index.php?id=download-center"
	// Local file path
	localFilePath := "walkera.html"
	// Local file data.
	var getData string
	// Check if the file exists.
	if !fileExists(localFilePath) {
		// Get the data and save it to the local file.
		getData = getDataFromURL(remoteAPIURL)
		// Save the data to the local file.
		appendAndWriteToFile(localFilePath, getData)
	}
	if fileExists(localFilePath) {
		getData = readAFileAsString(localFilePath)
	}
	// Get the data from the downloaded file.
	finalList := extractPDFUrls(getData)
	outputDir := "PDFs/" // Directory to store downloaded PDFs
	// Check if its exists.
	if !directoryExists(outputDir) {
		// Create the dir
		createDirectory(outputDir, 0o755)
	}
	// Remove double from slice.
	// finalList = removeDuplicatesFromSlice(finalList)
	// Get all the values.
	for _, urls := range finalList {
		if !strings.Contains(urls, "https://en.walkera.com/") {
			urls = "https://en.walkera.com/" + urls
		}
		// Check if the url is valid.
		if isUrlValid(urls) {
			// Download the pdf.
			downloadPDF(urls, outputDir)
		}
	}
}

// Read a file and return the contents
func readAFileAsString(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		log.Println(err)
	}
	return string(content)
}

// Append and write to file
func appendAndWriteToFile(path string, content string) {
	filePath, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	_, err = filePath.WriteString(content + "\n")
	if err != nil {
		log.Println(err)
	}
	err = filePath.Close()
	if err != nil {
		log.Println(err)
	}
}

// getFilename extracts the filename from a given file path.
func getFilename(path string) string {
	return filepath.Base(path) // Use filepath.Base() to get the last element (filename) from the path
}


// Converts a URL into a filesystem-safe PDF filename
func urlToFilename(rawURL string) string {
	// Convert the entire URL to lowercase to ensure consistency
	lower := strings.ToLower(rawURL)

	// Convert the url into a file name only.
	lower = getFilename(lower)

	// Compile a regex that matches any character that is not a lowercase letter or digit
	reNonAlnum := regexp.MustCompile(`[^a-z0-9]`)

	// Replace all non-alphanumeric characters with underscores
	safe := reNonAlnum.ReplaceAllString(lower, "_")

	// Collapse multiple consecutive underscores into a single underscore
	safe = regexp.MustCompile(`_+`).ReplaceAllString(safe, "_")

	// Remove any leading or trailing underscores
	safe = strings.Trim(safe, "_")

	// List of known unwanted substrings to remove from the filename
	var invalidSubstrings = []string{
		"_pdf", // Unwanted suffix/prefix that might already exist
	}

	// Remove each unwanted substring from the filename
	for _, invalidPre := range invalidSubstrings {
		safe = removeSubstring(safe, invalidPre)
	}

	// Append the ".pdf" extension if it's not already present
	if getFileExtension(safe) != ".pdf" {
		safe = safe + ".pdf"
	}

	// Return the final, sanitized filename
	return safe
}

// Removes all instances of a substring from a string
func removeSubstring(input string, toRemove string) string {
	result := strings.ReplaceAll(input, toRemove, "") // Replace with empty
	return result
}

// getFileExtension returns the file extension
func getFileExtension(path string) string {
	return filepath.Ext(path) // Use filepath to extract extension
}

// fileExists checks whether a file exists at the given path
func fileExists(filename string) bool {
	info, err := os.Stat(filename) // Get file info
	if err != nil {
		return false // Return false if file doesn't exist or error occurs
	}
	return !info.IsDir() // Return true if it's a file (not a directory)
}

// downloadPDF downloads a PDF from the given URL and saves it in the specified output directory.
// It uses a WaitGroup to support concurrent execution and returns true if the download succeeded.
func downloadPDF(finalURL, outputDir string) bool {
	// Sanitize the URL to generate a safe file name
	filename := strings.ToLower(urlToFilename(finalURL))

	// Construct the full file path in the output directory
	filePath := filepath.Join(outputDir, filename)

	// Skip if the file already exists
	if fileExists(filePath) {
		log.Printf("File already exists, skipping: %s", filePath)
		return false
	}

	// Create an HTTP client with a timeout
	client := &http.Client{Timeout: 15 * time.Minute}

	// Send GET request
	resp, err := client.Get(finalURL)
	if err != nil {
		log.Printf("Failed to download %s: %v", finalURL, err)
		return false
	}
	defer resp.Body.Close()

	// Check HTTP response status
	if resp.StatusCode != http.StatusOK {
		log.Printf("Download failed for %s: %s", finalURL, resp.Status)
		return false
	}

	// Check Content-Type header
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/pdf") {
		log.Printf("Invalid content type for %s: %s (expected application/pdf)", finalURL, contentType)
		return false
	}

	// Read the response body into memory first
	var buf bytes.Buffer
	written, err := io.Copy(&buf, resp.Body)
	if err != nil {
		log.Printf("Failed to read PDF data from %s: %v", finalURL, err)
		return false
	}
	if written == 0 {
		log.Printf("Downloaded 0 bytes for %s; not creating file", finalURL)
		return false
	}

	// Only now create the file and write to disk
	out, err := os.Create(filePath)
	if err != nil {
		log.Printf("Failed to create file for %s: %v", finalURL, err)
		return false
	}
	defer out.Close()

	if _, err := buf.WriteTo(out); err != nil {
		log.Printf("Failed to write PDF to file for %s: %v", finalURL, err)
		return false
	}

	log.Printf("Successfully downloaded %d bytes: %s → %s", written, finalURL, filePath)
	return true
}

// Checks if the directory exists
// If it exists, return true.
// If it doesn't, return false.
func directoryExists(path string) bool {
	directory, err := os.Stat(path)
	if err != nil {
		return false
	}
	return directory.IsDir()
}

// The function takes two parameters: path and permission.
// We use os.Mkdir() to create the directory.
// If there is an error, we use log.Println() to log the error and then exit the program.
func createDirectory(path string, permission os.FileMode) {
	err := os.Mkdir(path, permission)
	if err != nil {
		log.Println(err)
	}
}

// Checks whether a URL string is syntactically valid
func isUrlValid(uri string) bool {
	_, err := url.ParseRequestURI(uri) // Attempt to parse the URL
	return err == nil                  // Return true if no error occurred
}

// Remove all the duplicates from a slice and return the slice.
func removeDuplicatesFromSlice(slice []string) []string {
	check := make(map[string]bool)
	var newReturnSlice []string
	for _, content := range slice {
		if !check[content] {
			check[content] = true
			newReturnSlice = append(newReturnSlice, content)
		}
	}
	return newReturnSlice
}

// extractPDFUrls extracts all PDF URLs from <a href="..."> tags in the HTML.
func extractPDFUrls(htmlInput string) []string {
	var pdfLinks []string

	doc, err := html.Parse(strings.NewReader(htmlInput))
	if err != nil {
		log.Println(err)
		return nil
	}

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					href := strings.TrimSpace(attr.Val)
					if strings.Contains(strings.ToLower(href), ".pdf") {
						pdfLinks = append(pdfLinks, href)
					}
				}
			}
		}
		// Traverse child nodes
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(doc)
	return pdfLinks
}

// getDataFromURL performs an HTTP GET request and returns the response body as a string
func getDataFromURL(uri string) string {
	log.Println("Scraping", uri)   // Log the URL being scraped
	response, err := http.Get(uri) // Perform GET request
	if err != nil {
		log.Println(err) // Exit if request fails
	}

	body, err := io.ReadAll(response.Body) // Read response body
	if err != nil {
		log.Println(err) // Exit if read fails
	}

	err = response.Body.Close() // Close response body
	if err != nil {
		log.Println(err) // Exit if close fails
	}
	return string(body)
}
