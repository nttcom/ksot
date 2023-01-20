/*
 Copyright (c) 2022 NTT Communications Corporation

 Permission is hereby granted, free of charge, to any person obtaining a copy
 of this software and associated documentation files (the "Software"), to deal
 in the Software without restriction, including without limitation the rights
 to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 copies of the Software, and to permit persons to whom the Software is
 furnished to do so, subject to the following conditions:

 The above copyright notice and this permission notice shall be included in
 all copies or substantial portions of the Software.

 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 THE SOFTWARE.
*/

package artifact

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/fluxcd/pkg/untar"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/pkg/errors"
)

const (
	EnvSourceHost = "SOURCE_HOST"
)

func FetchArtifact(ctx context.Context, repository sourcev1.GitRepository, dir string) (string, error) {
	if repository.Status.Artifact == nil {
		return "", fmt.Errorf("repository %s does not contain an artifact", repository.Name)
	}

	url := repository.Status.Artifact.URL

	// for local run:
	// kubectl -n flux-system port-forward svc/source-controller 8080:80
	// export SOURCE_HOST=localhost:8080
	if hostname := os.Getenv(EnvSourceHost); hostname != "" {
		url = fmt.Sprintf("http://%s/gitrepository/%s/%s/latest.tar.gz", hostname, repository.Namespace, repository.Name)
	}

	if url == "" {
		return "", fmt.Errorf("no url given")
	}

	// download the tarball
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", errors.WithStack(fmt.Errorf("create HTTP request: %w", err))
	}

	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return "", errors.WithStack(fmt.Errorf("download artifact from %s: %w", url, err))
	}
	defer resp.Body.Close()

	// check response
	if resp.StatusCode != http.StatusOK {
		return "", errors.WithStack(fmt.Errorf("download artifact, status: %s", resp.Status))
	}

	var buf bytes.Buffer

	// verify checksum matches origin
	if err := verifyArtifact(repository.GetArtifact(), &buf, resp.Body); err != nil {
		return "", err
	}

	// extract
	summary, err := untar.Untar(&buf, dir)
	if err != nil {
		return "", errors.WithStack(fmt.Errorf("untar artifact: %w", err))
	}

	return summary, nil
}

func FetchArtifactAt(ctx context.Context, repository sourcev1.GitRepository, dir, revision string) (string, error) {
	if repository.Status.Artifact == nil {
		return "", fmt.Errorf("repository %s does not contain an artifact", repository.Name)
	}

	url := repository.Status.Artifact.URL
	if hostname := os.Getenv(EnvSourceHost); hostname != "" {
		url = fmt.Sprintf("http://%s/gitrepository/%s/%s/latest.tar.gz", hostname, repository.Namespace, repository.Name)
	}
	url = ReplaceRevision(url, revision)

	if url == "" {
		return "", fmt.Errorf("no url given")
	}

	// download the tarball
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", errors.WithStack(fmt.Errorf("create HTTP request: %w", err))
	}

	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return "", errors.WithStack(fmt.Errorf("download artifact from %s: %w", url, err))
	}
	defer resp.Body.Close()

	// check response
	if resp.StatusCode != http.StatusOK {
		return "", errors.WithStack(fmt.Errorf("download artifact, status: %s", resp.Status))
	}

	// extract
	summary, err := untar.Untar(resp.Body, dir)
	if err != nil {
		return "", errors.WithStack(fmt.Errorf("untar artifact: %w", err))
	}

	return summary, nil
}

func ReplaceRevision(url, revision string) string {
	re := regexp.MustCompile(`/(\w+).tar.gz$`)
	exRev := re.FindStringSubmatch(url)
	if exRev == nil {
		return url
	}
	return strings.ReplaceAll(url, exRev[1], revision)
}

func verifyArtifact(artifact *sourcev1.Artifact, buf *bytes.Buffer, reader io.Reader) error {
	hasher := sha256.New()

	// compute checksum
	mw := io.MultiWriter(hasher, buf)
	if _, err := io.Copy(mw, reader); err != nil {
		return errors.WithStack(err)
	}

	if checksum := fmt.Sprintf("%x", hasher.Sum(nil)); checksum != artifact.Checksum {
		return fmt.Errorf("verify artifact: computed checksum '%s' doesn't match advertised '%s'",
			checksum, artifact.Checksum)
	}

	return nil
}
