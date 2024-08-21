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

type ConstructFilenameFunction func(method string, requestUrl string, requestBody interface{}) (string, error)

type RecorderImpl struct {
	Wrapped               aurestclientapi.Client
	RecorderPath          string
	ConstructFilenameFunc ConstructFilenameFunction
}

const RecorderPathEnvVariable = "GO_AUTUMN_RESTCLIENT_RECORDER_PATH"

type RecorderOptions struct {
	ConstructFilenameFunc ConstructFilenameFunction
}

// New builds a new http recorder.
//
// Insert this into your stack just above the actual http client.
//
// Normally it does nothing, but if you set the environment variable RecorderPathEnvVariable to a path to a directory,
// it will write response recordings for your requests that you can then play back using aurestplayback.PlaybackImpl
// in your tests.
//
// You can optionally add a RecorderOptions instance to your call. The ... is really just so it's an optional argument.
func New(wrapped aurestclientapi.Client, additionalOptions ...RecorderOptions) aurestclientapi.Client {
	recorderPath, filenameFunc := initRecorderPathAndFilenameFunc(additionalOptions)
	return &RecorderImpl{
		Wrapped:               wrapped,
		RecorderPath:          recorderPath,
		ConstructFilenameFunc: filenameFunc,
	}
}

func initRecorderPathAndFilenameFunc(additionalOptions []RecorderOptions) (string, ConstructFilenameFunction) {
	recorderPath := os.Getenv(RecorderPathEnvVariable)
	if recorderPath != "" {
		if !strings.HasSuffix(recorderPath, "/") {
			recorderPath += "/"
		}
	}
	filenameFunc := ConstructFilenameV3WithBody
	for _, o := range additionalOptions {
		if o.ConstructFilenameFunc != nil {
			filenameFunc = o.ConstructFilenameFunc
		}
	}
	return recorderPath, filenameFunc
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

	recordResponseData(method, requestUrl, requestBody, response, responseErr, c.RecorderPath, c.ConstructFilenameFunc)
	return responseErr
}

func recordResponseData(method string, requestUrl string, requestBody interface{},
	response *aurestclientapi.ParsedResponse, responseErr error,
	recorderPath string, constructFilenameFunc ConstructFilenameFunction) {
	if recorderPath != "" {
		filename, err := constructFilenameFunc(method, requestUrl, requestBody)
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
				_ = os.WriteFile(recorderPath+filename, jsonRecording, 0644)
			}
		}
	}
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

func ConstructFilenameWithBody(method string, requestUrl string, _ interface{}) (string, error) {
	return ConstructFilename(method, requestUrl)
}

func ConstructFilenameV2(method string, requestUrl string) (string, error) {
	parsedUrl, err := url.Parse(requestUrl)
	if err != nil {
		return "", err
	}

	m := strings.ToLower(method)
	p := url.QueryEscape(parsedUrl.EscapedPath())
	if len(p) > 120 {
		p = string([]byte(p)[:120])
	}
	p = strings.ReplaceAll(p, "%2F", "-")
	p = strings.TrimLeft(p, "-")
	p = strings.TrimRight(p, "-")

	// we have to ensure the filenames don't get too long. git for windows only supports 260 character paths
	md5sumOverQuery := md5.Sum([]byte(parsedUrl.RawQuery))
	q := hex.EncodeToString(md5sumOverQuery[:])
	q = q[:8]

	filename := fmt.Sprintf("request_%s_%s_%s.json", m, p, q)
	return filename, nil
}

func ConstructFilenameV2WithBody(method string, requestUrl string, _ interface{}) (string, error) {
	return ConstructFilenameV2(method, requestUrl)
}

func ConstructFilenameV3(method string, requestUrl string) (string, error) {
	parsedUrl, err := url.Parse(requestUrl)
	if err != nil {
		return "", err
	}

	m := strings.ToLower(method)
	p := url.QueryEscape(parsedUrl.EscapedPath())
	if len(p) > 120 {
		p = string([]byte(p)[:120])
	}
	p = strings.ReplaceAll(p, "%2F", "-")
	p = strings.TrimLeft(p, "-")
	p = strings.TrimRight(p, "-")

	// we have to ensure the filenames don't get too long. git for windows only supports 260 character paths
	md5sumOverQuery := md5.Sum([]byte(parsedUrl.Query().Encode()))
	q := hex.EncodeToString(md5sumOverQuery[:])
	q = q[:8]

	filename := fmt.Sprintf("request_%s_%s_%s.json", m, p, q)
	return filename, nil
}

func ConstructFilenameV3WithBody(method string, requestUrl string, _ interface{}) (string, error) {
	return ConstructFilenameV3(method, requestUrl)
}
