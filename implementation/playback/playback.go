package aurestplayback

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	aulogging "github.com/StephanHCB/go-autumn-logging"
	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
	aurestrecorder "github.com/StephanHCB/go-autumn-restclient/implementation/recorder"
)

const PlaybackRewritePathEnvVariable = "GO_AUTUMN_RESTCLIENT_PLAYBACK_REWRITE_PATH"

type PlaybackImpl struct {
	RecorderPath                string
	RecorderRewritePath         string
	ConstructFilenameCandidates []aurestrecorder.ConstructFilenameFunction
	Now                         func() time.Time
}

type PlaybackOptions struct {
	// ConstructFilenameCandidates contains filename constructor functions.
	//
	// The first one is considered "canonical", for all others, a log entry is printed that instructs the user
	// to rename the file.
	ConstructFilenameCandidates []aurestrecorder.ConstructFilenameFunction
	NowFunc                     func() time.Time
}

// New builds a new http client simulator based on playback.
//
// Use this in your tests.
//
// You can optionally add a PlaybackOptions instance to your call. The ... is really just so it's an optional argument.
func New(recorderPath string, additionalOptions ...PlaybackOptions) aurestclientapi.Client {
	recorderRewritePath, filenameCandidates, nowFunc := initRecorderPathAndFilenameFunc(additionalOptions)

	return &PlaybackImpl{
		RecorderPath:                recorderPath,
		RecorderRewritePath:         recorderRewritePath,
		ConstructFilenameCandidates: filenameCandidates,
		Now:                         nowFunc,
	}
}

func initRecorderPathAndFilenameFunc(additionalOptions []PlaybackOptions) (string, []aurestrecorder.ConstructFilenameFunction, func() time.Time) {
	filenameCandidates := []aurestrecorder.ConstructFilenameFunction{
		aurestrecorder.ConstructFilenameV3WithBody,
		aurestrecorder.ConstructFilenameWithBody,
		aurestrecorder.ConstructFilenameV2WithBody,
	}
	nowFunc := time.Now
	for _, o := range additionalOptions {
		if len(o.ConstructFilenameCandidates) > 0 {
			filenameCandidates = o.ConstructFilenameCandidates
		}
		if o.NowFunc != nil {
			nowFunc = o.NowFunc
		}
	}
	recorderRewritePath := os.Getenv(PlaybackRewritePathEnvVariable)
	return recorderRewritePath, filenameCandidates, nowFunc
}

func (c *PlaybackImpl) Perform(ctx context.Context, method string, requestUrl string, requestBody interface{}, response *aurestclientapi.ParsedResponse) error {
	canonicalFilename := ""
	var originalError error
	for i, constructFilenameCandidate := range c.ConstructFilenameCandidates {
		filename, err := constructFilenameCandidate(method, requestUrl, requestBody)
		if err != nil {
			return err
		}
		if i == 0 {
			canonicalFilename = filename
		}

		jsonBytes, err := os.ReadFile(filepath.Join(c.RecorderPath, filename))
		if err != nil {
			if i == 0 {
				originalError = err
			}
		} else {
			// successfully read
			if i > 0 {
				aulogging.Logger.Ctx(ctx).Info().Printf("use of deprecated recorder filename '%s', please move to '%s'", filename, canonicalFilename)
			}

			_ = c.rewriteFileIfConfigured(ctx, filename, canonicalFilename)

			recording := aurestrecorder.RecorderData{}
			err = json.Unmarshal(jsonBytes, &recording)
			if err != nil {
				return err
			}

			response.Header = recording.ParsedResponse.Header
			response.Status = recording.ParsedResponse.Status
			response.Time = c.Now()

			switch response.Body.(type) {
			case **[]byte:
				asString, ok := recording.ParsedResponse.Body.(string)
				if ok {
					asBytes := []byte(asString)
					*(response.Body.(**[]byte)) = &asBytes
				} else {
					// For backwards compatibility with existing recordings we fall back to the previous logic.
					// This is because in these old recordings the body is stored as a json object instead of a string.
					// This is not compatible with the changes introduced in 0.7.2
					bodyJsonBytes, err := json.Marshal(recording.ParsedResponse.Body)
					if err != nil {
						return err
					}
					*(response.Body.(**[]byte)) = &bodyJsonBytes
				}
			default:
				// cannot just assign the body, need to re-parse into the existing pointer - using a json round trip
				bodyJsonBytes, err := json.Marshal(recording.ParsedResponse.Body)
				if err != nil {
					return err
				}
				err = json.Unmarshal(bodyJsonBytes, response.Body)
				if err != nil {
					return err
				}
			}

			return recording.Error
		}
	}
	return originalError
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
