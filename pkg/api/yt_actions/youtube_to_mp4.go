package yt_actions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
)

// credits to https://github.com/apchavan/go-youtube-downloader/blob/main/runner/network_ops.go#L61

type ClientRequestMapStruct struct {
	Hl                string `json:"hl"`
	Gl                string `json:"gl"`
	ClientName        string `json:"clientName"`
	ClientVersion     string `json:"clientVersion"`
	ClientScreen      string `json:"clientScreen"`
	AndroidSdkVersion int    `json:"androidSdkVersion"`
}

type ThirdPartyRequestMapStruct struct {
	EmbedUrl string `json:"embedUrl"`
}

type ContextRequestMapStruct struct {
	Client     ClientRequestMapStruct     `json:"client"`
	ThirdParty ThirdPartyRequestMapStruct `json:"thirdParty"`
}

type RequestBodyStruct struct {
	Context ContextRequestMapStruct `json:"context"`

	VideoId        string `json:"videoId"`
	RacyCheckOk    bool   `json:"racyCheckOk"`
	ContentCheckOk bool   `json:"contentCheckOk"`
}

func HandleYTMP4Routes(router *gin.Engine) {
	router.POST("/api/handleYTMP4", HandleTYMP4)
}

func extractVideoID(url string) (string, error) {
	// Example: Extract video ID from a YouTube URL
	regexPattern := `(?:youtube\.com\/\S*(?:(?:\/e(?:mbed))?\/|watch\?(?:\S*?&?v=))|youtu\.be\/)([a-zA-Z0-9_-]{11})`
	regex := regexp.MustCompile(regexPattern)
	matches := regex.FindStringSubmatch(url)
	if len(matches) > 1 {
		return matches[1], nil
	}

	return "", nil
}
func createHTTPClient(proxyStr string) (*http.Client, error) {
	//implement in nessescary
	return &http.Client{}, nil
}

func GetVideoMetadataFromYouTubei(videoURL string) (map[string]interface{}, error) {
	videoID, err := extractVideoID(videoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to extract video ID: %v", err)
	}

	client := &http.Client{}

	requestContext := map[string]interface{}{
		"context": map[string]interface{}{
			"client": map[string]interface{}{
				"hl":               "en",
				"gl":               "US",
				"clientName":       "WEB",
				"clientVersion":    "2.20230221.01.00",
				"platform":         "DESKTOP",
				"userAgent":        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/110.0.0.0 Safari/537.36",
				"visitorData":      "Cg0KC3Rlc3RfVklTSVNfRmlsZQ==", // Added visitorData
				"timeZone":         "UTC",
				"utcOffsetMinutes": 0,
			},
			"user": map[string]interface{}{
				"lockedSafetyMode": false,
			},
		},
		"videoId":        videoID,
		"racyCheckOk":    true,
		"contentCheckOk": true,
		"playbackContext": map[string]interface{}{
			"contentPlaybackContext": map[string]interface{}{
				"html5Preference": "HTML5_PREF_WANTS",
			},
		},
	}

	reqJSON, err := json.Marshal(requestContext)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	youtubeiURL := "https://www.youtube.com/youtubei/v1/player?key=AIzaSyAO_FJ2SlqU8Q4STEHLGCilw_Y9_11qcW8"
	req, err := http.NewRequest("POST", youtubeiURL, bytes.NewReader(reqJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// set headers to mimic a real browser request (pls don't ban me youtube)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/110.0.0.0 Safari/537.36")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Origin", "https://www.youtube.com")
	req.Header.Set("Referer", "https://www.youtube.com/watch?v="+videoID)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("X-YouTube-Client-Name", "1")
	req.Header.Set("X-YouTube-Client-Version", "2.20230221.01.00")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad response status: %s", resp.Status)
	}

	var responseData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return responseData, nil
}

func downloadYTContent(downloadURL string) error {

	// streamingData, err := metadata["streamingData"].(map[string]interface{})
	//look for something in the thing for a url then download

	return fmt.Errorf("not implemented yet")
}

func HandleTYMP4(c *gin.Context) {
	url := c.PostForm("url")
	body_real, err := GetVideoMetadataFromYouTubei(url)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to get video metadata: %v", err),
		})
		return
	}

	fmt.Print("Handling YTMP4 request\n")
	c.JSON(200, gin.H{

		"message": body_real,
		"test":    "test",
	})

	//	c.FileAttachment(outputPath, "rotated_"+filepath.Base(filePath)[9:]) return like this as a download
}
