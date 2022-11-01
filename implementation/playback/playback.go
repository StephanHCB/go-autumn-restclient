package aurestplayback

import (
	"context"
	"encoding/json"
	aulogging "github.com/StephanHCB/go-autumn-logging"
	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
	aurestrecorder "github.com/StephanHCB/go-autumn-restclient/implementation/recorder"
	"os"
	"strings"
	"time"
)

type PlaybackImpl struct {
	RecorderPath string
	// Now is exposed so tests can fixate the time by overwriting this field
	Now func() time.Time
}

// New builds a new http client simulator based on playback.
//
// Use this in your tests.
func New(recorderPath string) aurestclientapi.Client {
	if recorderPath != "" {
		if !strings.HasSuffix(recorderPath, "/") {
			recorderPath += "/"
		}
	}
	return &PlaybackImpl{
		RecorderPath: recorderPath,
		Now:          time.Now,
	}
}

func (c *PlaybackImpl) Perform(ctx context.Context, method string, requestUrl string, _ interface{}, response *aurestclientapi.ParsedResponse) error {
	filename, err := aurestrecorder.ConstructFilenameV3(method, requestUrl)
	if err != nil {
		return err
	}

	jsonBytes, err := os.ReadFile(c.RecorderPath + filename)
	if err != nil {
		// try old filename for compatibility (cannot fail if ConstructFilenameV2 didn't)
		filenameOldV1, _ := aurestrecorder.ConstructFilename(method, requestUrl)

		jsonBytesOldV1, errWithOldFilenameV1 := os.ReadFile(c.RecorderPath + filenameOldV1)
		if errWithOldFilenameV1 != nil {
			// try old filename for compatibility (cannot fail if ConstructFilenameV2 didn't)
			filenameOldV2, _ := aurestrecorder.ConstructFilenameV2(method, requestUrl)

			jsonBytesOldV2, errWithOldFilenameV2 := os.ReadFile(c.RecorderPath + filenameOldV2)
			if errWithOldFilenameV2 != nil {
				// but return original error if that also fails
				return err
			} else {
				aulogging.Logger.Ctx(ctx).Info().Printf("use of deprecated recorder filename (v2) '%s', please move to '%s'", filenameOldV2, filename)
				filename = filenameOldV2
				jsonBytes = jsonBytesOldV2
			}
		} else {
			aulogging.Logger.Ctx(ctx).Info().Printf("use of deprecated recorder filename (v1) '%s', please move to '%s'", filenameOldV1, filename)
			filename = filenameOldV1
			jsonBytes = jsonBytesOldV1
		}
	}

	recording := aurestrecorder.RecorderData{}
	err = json.Unmarshal(jsonBytes, &recording)
	if err != nil {
		return err
	}

	response.Header = recording.ParsedResponse.Header
	response.Status = recording.ParsedResponse.Status
	response.Time = c.Now()

	// cannot just assign the body, need to re-parse into the existing pointer - using a json round trip
	bodyJsonBytes, err := json.Marshal(recording.ParsedResponse.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(bodyJsonBytes, response.Body)
	if err != nil {
		return err
	}

	return recording.Error
}
