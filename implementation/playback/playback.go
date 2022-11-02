package aurestplayback

import (
	"context"
	"encoding/json"
	aulogging "github.com/StephanHCB/go-autumn-logging"
	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
	aurestrecorder "github.com/StephanHCB/go-autumn-restclient/implementation/recorder"
	"os"
	"path/filepath"
	"time"
)

const PlaybackRewritePathEnvVariable = "GO_AUTUMN_RESTCLIENT_PLAYBACK_REWRITE_PATH"

type PlaybackImpl struct {
	RecorderPath        string
	RecorderRewritePath string
	// Now is exposed so tests can fixate the time by overwriting this field
	Now func() time.Time
}

// New builds a new http client simulator based on playback.
//
// Use this in your tests.
func New(recorderPath string) aurestclientapi.Client {
	return &PlaybackImpl{
		RecorderPath:        recorderPath,
		RecorderRewritePath: os.Getenv(PlaybackRewritePathEnvVariable),
		Now:                 time.Now,
	}
}

func (c *PlaybackImpl) Perform(ctx context.Context, method string, requestUrl string, _ interface{}, response *aurestclientapi.ParsedResponse) error {
	filename, err := aurestrecorder.ConstructFilenameV3(method, requestUrl)
	if err != nil {
		return err
	}

	jsonBytes, err := os.ReadFile(filepath.Join(c.RecorderPath, filename))
	if err != nil {
		// try old filename for compatibility (cannot fail if ConstructFilenameV2 didn't)
		filenameOldV1, _ := aurestrecorder.ConstructFilename(method, requestUrl)

		jsonBytesOldV1, errWithOldFilenameV1 := os.ReadFile(filepath.Join(c.RecorderPath, filenameOldV1))
		if errWithOldFilenameV1 != nil {
			// try old filename for compatibility (cannot fail if ConstructFilenameV2 didn't)
			filenameOldV2, _ := aurestrecorder.ConstructFilenameV2(method, requestUrl)

			jsonBytesOldV2, errWithOldFilenameV2 := os.ReadFile(filepath.Join(c.RecorderPath, filenameOldV2))
			if errWithOldFilenameV2 != nil {
				// but return original error if that also fails
				return err
			} else {
				aulogging.Logger.Ctx(ctx).Info().Printf("use of deprecated recorder filename (v2) '%s', please move to '%s'", filenameOldV2, filename)

				_ = c.rewriteFileIfConfigured(ctx, filenameOldV2, filename)

				filename = filenameOldV2
				jsonBytes = jsonBytesOldV2
			}
		} else {
			aulogging.Logger.Ctx(ctx).Info().Printf("use of deprecated recorder filename (v1) '%s', please move to '%s'", filenameOldV1, filename)

			_ = c.rewriteFileIfConfigured(ctx, filenameOldV1, filename)

			filename = filenameOldV1
			jsonBytes = jsonBytesOldV1
		}
	} else {
		_ = c.rewriteFileIfConfigured(ctx, filename, filename)
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

func (c *PlaybackImpl) rewriteFileIfConfigured(ctx context.Context, fileNameFrom string, fileNameTo string) error {
	if c.RecorderRewritePath != "" {
		fileBase := filepath.Base(c.RecorderPath)

		bytes, err := os.ReadFile(filepath.Join(c.RecorderPath, fileNameFrom))
		if err != nil {
			aulogging.Logger.Ctx(ctx).Error().WithErr(err).Printf("Can't read file: '%s'", fileNameFrom)
			return err
		}

		targetPath := filepath.Join(c.RecorderRewritePath, fileBase)
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			_ = os.MkdirAll(targetPath, 0700)
		}

		err = os.WriteFile(filepath.Join(targetPath, fileNameTo), bytes, 0644)
		if err != nil {
			aulogging.Logger.Ctx(ctx).Error().WithErr(err).Printf("Can't write file: '%s'", fileNameTo)
			return err
		}
	}
	return nil
}
