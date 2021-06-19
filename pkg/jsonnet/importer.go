/*
Copyright 2017 The kubecfg authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.

Copyright 2021 Pelotech - Apache License, Version 2.0.
  - Adaptation for latest go-jsonnet and goembed
  - Adds HTTP caching
*/

package jsonnet

import (
	"embed"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/go-jsonnet"
)

//go:embed lib
var internalLib embed.FS

// MakeUniversalImporter returns an importer that can handle filepaths, HTTP urls, and internal paths.
func MakeUniversalImporter(log logr.Logger, searchURLs []*url.URL, cacheDir string) jsonnet.Importer {
	// Reconstructed copy of http.DefaultTransport (to avoid
	// modifying the default)
	t := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	t.RegisterProtocol("file", http.NewFileTransport(http.Dir("/")))
	t.RegisterProtocol("internal", http.NewFileTransport(http.FS(internalLib)))

	return &universalImporter{
		BaseSearchURLs: searchURLs,
		HTTPCache:      NewHTTPCache(log, t, cacheDir),
		cache:          map[string]jsonnet.Contents{},
	}
}

type universalImporter struct {
	BaseSearchURLs []*url.URL
	HTTPCache      *httpCache
	cache          map[string]jsonnet.Contents
}

func (importer *universalImporter) Import(importedFrom, importedPath string) (jsonnet.Contents, string, error) {
	candidateURLs, err := importer.expandImportToCandidateURLs(importedFrom, importedPath)
	if err != nil {
		return jsonnet.Contents{}, "", fmt.Errorf("could not get candidate URLs for when importing %s (imported from %s): %v", importedPath, importedFrom, err)
	}

	var tried []string
	for _, u := range candidateURLs {
		foundAt := u.String()
		if c, ok := importer.cache[foundAt]; ok {
			return c, foundAt, nil
		}

		tried = append(tried, foundAt)
		importedData, err := importer.HTTPCache.Get(foundAt)
		if err == nil {
			importer.cache[foundAt] = importedData
			return importedData, foundAt, nil
		} else if err != errNotFound {
			return jsonnet.Contents{}, "", err
		}
	}

	return jsonnet.Contents{}, "", fmt.Errorf("couldn't open import %q, no match locally or in library search paths. Tried: %s",
		importedPath,
		strings.Join(tried, ";"),
	)
}

func (importer *universalImporter) expandImportToCandidateURLs(importedFrom, importedPath string) ([]*url.URL, error) {
	importedPathURL, err := url.Parse(importedPath)
	if err != nil {
		return nil, fmt.Errorf("import path %q is not valid", importedPath)
	}
	if importedPathURL.IsAbs() {
		return []*url.URL{importedPathURL}, nil
	}

	importDirURL, err := url.Parse(importedFrom)
	if err != nil {
		return nil, fmt.Errorf("invalid import dir %q: %v", importedFrom, err)
	}

	candidateURLs := make([]*url.URL, 1, len(importer.BaseSearchURLs)+1)
	candidateURLs[0] = importDirURL.ResolveReference(importedPathURL)

	for _, u := range importer.BaseSearchURLs {
		candidateURLs = append(candidateURLs, u.ResolveReference(importedPathURL))
	}

	return candidateURLs, nil
}
