/*
Copyright 2021 Pelotech.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package jsonnet

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-logr/logr"
	jsonnet "github.com/google/go-jsonnet"
)

var errNotFound = errors.New("not found")

// cache implements a dumb local cache for files fetched remotely.
type httpCache struct {
	// The location of the cache directory
	cacheDir string
	// The http client used for requests
	httpClient *http.Client
	// The logger for the cache
	log logr.Logger
}

// NewHTTPCache creates a new http cache using the given transport and cache directory.
func NewHTTPCache(log logr.Logger, t *http.Transport, cacheDir string) *httpCache {
	return &httpCache{
		cacheDir: cacheDir,
		httpClient: &http.Client{
			Transport: t,
		},
		log: log,
	}
}

var httpRegex = regexp.MustCompile("^(https?)://")
var internalRegex = regexp.MustCompile("^internal:///?(.*)$")

func (h *httpCache) getLocalPath(url string) string {
	return filepath.Join(h.cacheDir, httpRegex.ReplaceAllString(url, ""))
}

func (h *httpCache) tryLocalCache(url string) (jsonnet.Contents, error) {
	localPath := h.getLocalPath(url)
	bytes, err := ioutil.ReadFile(localPath)
	if err != nil {
		return jsonnet.Contents{}, err
	}
	return jsonnet.MakeContents(string(bytes)), nil
}

func (h *httpCache) writeToCache(url string, contents []byte) error {
	localPath := h.getLocalPath(url)
	localPathDir := filepath.Dir(localPath)
	finfo, err := os.Stat(localPathDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(localPathDir, 0755); err != nil {
			return err
		}
	}
	if err == nil && !finfo.IsDir() {
		return fmt.Errorf("%q is not a directory, it cannot be used for caching", localPathDir)
	}
	return ioutil.WriteFile(localPath, contents, 0644)
}

func (h *httpCache) Get(url string) (jsonnet.Contents, error) {
	isHTTP := httpRegex.MatchString(url)
	isInternal := internalRegex.MatchString(url)

	// If this is an http url, try the local cache first
	if isHTTP {
		contents, err := h.tryLocalCache(url)
		if err == nil {
			return contents, nil
		}
	}

	// If this is an internal URL make sure it is rooted at /lib
	if isInternal {
		match := internalRegex.FindStringSubmatch(url)
		if len(match) != 2 {
			return jsonnet.Contents{}, errors.New("malformed internal URL")
		}
		url = fmt.Sprintf("internal:///lib/%s", strings.TrimLeft(match[1], "lib/"))
		if strings.HasSuffix(url, "kubecfg.libsonnet") {
			url = "internal:///lib/kubecfg.libsonnet"
		}
	}

	// Attempt a normal GET
	res, err := h.httpClient.Get(url)
	if err != nil {
		return jsonnet.Contents{}, err
	}
	defer res.Body.Close()

	if isHTTP {
		h.log.Info(fmt.Sprintf("GET %q -> %s", url, res.Status))
	}
	if res.StatusCode == http.StatusNotFound {
		return jsonnet.Contents{}, errNotFound
	} else if res.StatusCode != http.StatusOK {
		return jsonnet.Contents{}, fmt.Errorf("error reading content: %s", res.Status)
	}

	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return jsonnet.Contents{}, err
	}

	// If it was an http url, write the contents to the local cache
	if isHTTP {
		if err := h.writeToCache(url, bodyBytes); err != nil {
			h.log.Error(err, fmt.Sprintf("Error writing %q to the local cache", url))
		}
	}

	return jsonnet.MakeContents(string(bodyBytes)), nil
}
