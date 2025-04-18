package itests

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/ory/dockertest/v3"
	dc "github.com/ory/dockertest/v3/docker"
)

const dockerfilePath = "./../Dockerfile.gowaves-it"
const (
	keepDanglingEnvKey     = "ITESTS_KEEP_DANGLING"
	withRaceDetectorEnvKey = "ITESTS_WITH_RACE_DETECTOR"
)

const (
	withRaceDetectorSuffixArgumentName  = "WITH_RACE_SUFFIX"
	withRaceDetectorSuffixArgumentValue = "-with-race"
)

func TestMain(m *testing.M) {
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get pwd: %v", err)
	}
	var (
		keepDangling     = mustBoolEnv(keepDanglingEnvKey)
		withRaceDetector = mustBoolEnv(withRaceDetectorEnvKey)
	)
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Failed to create docker pool: %v", err)
	}
	if plErr := pool.Client.PullImage(
		dc.PullImageOptions{
			Repository: "wavesplatform/wavesnode",
			Tag:        "latest",
			Platform:   "linux/amd64"},
		dc.AuthConfiguration{}); plErr != nil {
		log.Fatalf("Failed to pull node image: %v", plErr)
	}
	var buildArgs []dc.BuildArg
	if withRaceDetector {
		buildArgs = append(buildArgs, dc.BuildArg{
			Name: withRaceDetectorSuffixArgumentName, Value: withRaceDetectorSuffixArgumentValue,
		})
	}
	dir, file := filepath.Split(filepath.Join(pwd, dockerfilePath))
	err = pool.Client.BuildImage(dc.BuildImageOptions{
		Name:           "go-node",
		Dockerfile:     file,
		ContextDir:     dir,
		OutputStream:   io.Discard,
		BuildArgs:      buildArgs,
		Platform:       "",
		RmTmpContainer: true,
	})
	if err != nil {
		log.Fatalf("Failed to create go-node image: %v", err)
	}

	if !keepDangling { // remove dangling images
		images, lsErr := pool.Client.ListImages(dc.ListImagesOptions{
			Filters: map[string][]string{
				"label": {"wavesplatform-gowaves-itests-tmp=true"},
			},
		})
		if lsErr != nil {
			log.Fatalf("Failed to get list of images from docker: %v", lsErr)
		}
		for _, i := range images {
			rmErr := pool.Client.RemoveImageExtended(i.ID, dc.RemoveImageOptions{
				Force:   true,
				NoPrune: false,
				Context: nil,
			})
			if rmErr != nil {
				log.Fatalf("Failed to remove dangling images: %v", rmErr)
			}
		}
	}
	os.Exit(m.Run())
}

func mustBoolEnv(key string) bool {
	envFlag := os.Getenv(key)
	if envFlag == "" {
		return false
	}
	r, err := strconv.ParseBool(envFlag)
	if err != nil {
		log.Fatalf("Invalid flag value %q for the env key %q: %v", envFlag, key, err)
	}
	return r
}
