package aurestplayback

import (
	"context"
	"encoding/json"
	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
	aurestrecorder "github.com/StephanHCB/go-autumn-restclient/implementation/recorder"
	"os"
)

type PlaybackImpl struct {
	RecorderPath string
}

// New builds a new http client simulator based on playback.
//
// Use this in your tests.
func New(recorderPath string) aurestclientapi.Client {
	return &PlaybackImpl{
		RecorderPath: recorderPath,
	}
}

func (c *PlaybackImpl) Perform(_ context.Context, method string, requestUrl string, _ interface{}, response *aurestclientapi.ParsedResponse) error {
	filename, err := aurestrecorder.ConstructFilename(method, requestUrl)
	if err != nil {
		return err
	}

	jsonBytes, err := os.ReadFile(c.RecorderPath + filename)
	if err != nil {
		return err
	}

	recording := aurestrecorder.RecorderData{}
	err = json.Unmarshal(jsonBytes, &recording)
	if err != nil {
		return err
	}

	response.Header = recording.ParsedResponse.Header
	response.Status = recording.ParsedResponse.Status
	response.Body = recording.ParsedResponse.Body
	return recording.Error
}
