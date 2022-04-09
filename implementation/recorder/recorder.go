package aurestrecorder

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
	"net/url"
	"os"
	"strings"
)

type RecorderImpl struct {
	Wrapped      aurestclientapi.Client
	RecorderPath string
}

const RecorderPathEnvVariable = "GO_AUTUMN_RESTCLIENT_RECORDER_PATH"

// New builds a new http recorder.
//
// Insert this into your stack just above the actual http client.
//
// Normally it does nothing, but if you set the environment variable RecorderPathEnvVariable to a path to a directory,
// it will write response recordings for your requests that you can then play back using aurestplayback.PlaybackImpl
// in your tests.
func New(wrapped aurestclientapi.Client) aurestclientapi.Client {
	recorderPath := os.Getenv(RecorderPathEnvVariable)
	if recorderPath != "" {
		if !strings.HasSuffix(recorderPath, "/") {
			recorderPath += "/"
		}
	}
	return &RecorderImpl{
		Wrapped:      wrapped,
		RecorderPath: recorderPath,
	}
}

type RecorderData struct {
	Method         string                         `json:"method"`
	RequestUrl     string                         `json:"requestUrl"`
	RequestBody    interface{}                    `json:"requestBody,omitempty"`
	ParsedResponse aurestclientapi.ParsedResponse `json:"parsedResponse"`
	Error          error                          `json:"error,omitempty"`
}

func (c *RecorderImpl) Perform(ctx context.Context, method string, requestUrl string, requestBody interface{}, response *aurestclientapi.ParsedResponse) error {
	responseErr := c.Wrapped.Perform(ctx, method, requestUrl, requestBody, response)
	if c.RecorderPath != "" {
		filename, err := ConstructFilename(method, requestUrl)
		if err == nil {
			recording := RecorderData{
				Method:         method,
				RequestUrl:     requestUrl,
				RequestBody:    requestBody,
				ParsedResponse: *response,
				Error:          responseErr,
			}

			jsonRecording, err := json.MarshalIndent(&recording, "", "    ")
			if err == nil {
				_ = os.WriteFile(c.RecorderPath+filename, jsonRecording, 0644)
			}
		}
	}
	return responseErr
}

func ConstructFilename(method string, requestUrl string) (string, error) {
	parsedUrl, err := url.Parse(requestUrl)
	if err != nil {
		return "", err
	}

	m := strings.ToLower(method)
	p := url.QueryEscape(parsedUrl.EscapedPath())
	if len(p) > 120 {
		p = string([]byte(p)[:120])
	}
	// we have to ensure the filenames don't get too long. git for windows only supports 260 character paths
	md5sumOverQuery := md5.Sum([]byte(parsedUrl.RawQuery))
	q := hex.EncodeToString(md5sumOverQuery[:])

	filename := fmt.Sprintf("request_%s_%s_%s.json", m, p, q)
	return filename, nil
}
