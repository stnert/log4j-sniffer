// Copyright (c) 2021 Palantir Technologies. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package crawl_test

import (
	"context"
	"errors"
	"io/fs"
	"testing"
	"time"

	"github.com/palantir/log4j-sniffer/pkg/crawl"
	"github.com/palantir/log4j-sniffer/pkg/testcontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTgzIdentifierImplementsTimeout(t *testing.T) {
	identify := crawl.NewIdentifier(0, nil, func(ctx context.Context, path string) ([]string, error) {
		time.Sleep(50 * time.Millisecond)
		select {
		case <-ctx.Done():
			return nil, errors.New("Context was Cancelled")
		default:
			require.FailNow(t, "context should have been cancelled")
		}
		return nil, nil
	})
	_, _, err := identify.Identify(testcontext.GetTestContext(t), "", stubDirEntry{
		name: "sdlkfjsldkjfs.tar.gz",
	})
	assert.EqualError(t, err, "Context was Cancelled")
}

func TestZipIdentifierImplementsTimeout(t *testing.T) {
	identify := crawl.NewIdentifier(0, func(ctx context.Context, path string) ([]string, error) {
		time.Sleep(50 * time.Millisecond)
		select {
		case <-ctx.Done():
			return nil, errors.New("Context was Cancelled")
		default:
			require.FailNow(t, "context should have been cancelled")
		}
		return nil, nil
	}, nil)
	_, _, err := identify.Identify(testcontext.GetTestContext(t), "", stubDirEntry{
		name: "sdlkfjsldkjfs.zip",
	})
	assert.EqualError(t, err, "Context was Cancelled")
}

func TestIdentifyFromFileName(t *testing.T) {
	for _, tc := range []struct {
		name    string
		in      string
		result  crawl.Finding
		version string
	}{{
		name: "empty filename",
	}, {
		name: "plain filename",
		in:   "foo",
	}, {
		name:    "log4j x.y.z vulnerable version",
		in:      "log4j-core-2.10.0.jar",
		result:  crawl.JarName,
		version: "2.10.0",
	}, {
		name: "invalid file extension",
		in:   "log4j-core-2.10.0.png",
	}, {
		name:    "log4j patched version",
		in:      "log4j-core-2.16.0.jar",
		result:  crawl.JarName,
		version: "2.16.0",
	}, {
		name:    "log4j major minor vulnerable version",
		in:      "log4j-core-2.15.jar",
		result:  crawl.JarName,
		version: "2.15",
	}, {
		name: "log4j name not as start of filename",
		in:   "asdsadlog4j-core-2.14.1.jar",
	}, {
		name: "log4j name not as end of filename",
		in:   "log4j-core-2.14.1.jarf",
	}, {
		name:    "vulnerable release candidate",
		in:      "log4j-core-2.14.1-rc1.jar",
		result:  crawl.JarName,
		version: "2.14.1-rc1",
	}, {
		name:    "case-insensitive match",
		in:      "lOg4J-cOrE-2.14.0.jAr",
		result:  crawl.JarName,
		version: "2.14.0",
	}} {
		t.Run(tc.name, func(t *testing.T) {
			identify := crawl.NewIdentifier(time.Second, func(ctx context.Context, path string) ([]string, error) {
				// this is called for jars that are not identified as log4j
				// these cases are tested elsewhere so we just return nil with no error her
				return nil, nil
			}, panicOnList)
			result, version, err := identify.Identify(testcontext.GetTestContext(t), "", stubDirEntry{
				name: tc.in,
			})
			require.NoError(t, err)
			assert.Equal(t, tc.result, result)
			var expectedVersion string
			if tc.version == "" {
				expectedVersion = crawl.UnknownVersion
			} else {
				expectedVersion = tc.version
			}
			assert.Equal(t, expectedVersion, version)
		})
	}
}

func TestIdentifyFromZipContents(t *testing.T) {
	t.Run("handles error", func(t *testing.T) {
		expectedErr := errors.New("err")
		identify := crawl.NewIdentifier(time.Second, func(ctx context.Context, path string) ([]string, error) {
			assert.Equal(t, "/path/on/disk/", path)
			return nil, expectedErr
		}, panicOnList)
		_, _, err := identify.Identify(testcontext.GetTestContext(t), "/path/on/disk/", stubDirEntry{
			name: "file.zip",
		})
		require.Equal(t, expectedErr, err)
	})

	for _, tc := range []struct {
		name     string
		filename string
		zipList  []string
		tarList  []string
		result   crawl.Finding
		version  string
	}{{
		name:     "archive with no log4j",
		filename: "file.zip",
		zipList:  []string{"foo.jar"},
	}, {
		name:     "archive with vulnerable log4j version",
		filename: "file.zip",
		zipList:  []string{"foo.jar", "log4j-core-2.14.1.jar"},
		result:   crawl.JarNameInsideArchive,
		version:  "2.14.1",
	}, {
		name:     "tarred and gzipped with vulnerable log4j version",
		filename: "file.tar.gz",
		tarList:  []string{"foo.jar", "log4j-core-2.14.1.jar"},
		result:   crawl.JarNameInsideArchive,
		version:  "2.14.1",
	}, {
		name:     "tarred and gzipped with vulnerable log4j version, multiple . in filename",
		filename: "foo.bar.tar.gz",
		tarList:  []string{"foo.jar", "log4j-core-2.14.1.jar"},
		result:   crawl.JarNameInsideArchive,
		version:  "2.14.1",
	}, {
		name:     "archive with JndiLookup class in wrong package",
		filename: "java.jar",
		zipList:  []string{"a/package/with/JndiLookup.class"},
		result:   crawl.ClassName,
	}, {
		name:     "non-log4j archive with JndiLookup in the log4j package",
		filename: "not-log4.jar",
		zipList:  []string{"org/apache/logging/log4j/core/lookup/JndiLookup.class"},
		result:   crawl.ClassPackageAndName,
	}, {
		name:     "log4j named jar with JndiLookip class",
		filename: "log4j-core-2.14.1.jar",
		zipList:  []string{"org/apache/logging/log4j/core/lookup/JndiLookup.class"},
		result:   crawl.JarName | crawl.ClassPackageAndName,
		version:  "2.14.1",
	}, {
		name:     "zip with uppercase log4j inside",
		filename: "foo.jar",
		zipList:  []string{"log4j-core-2.14.1.jAr"},
		result:   crawl.JarNameInsideArchive,
		version:  "2.14.1",
	}} {
		t.Run(tc.name, func(t *testing.T) {
			identify := crawl.NewIdentifier(time.Second, func(ctx context.Context, path string) ([]string, error) {
				assert.Equal(t, "/path/on/disk/", path)
				return tc.zipList, nil
			}, func(ctx context.Context, path string) ([]string, error) {
				assert.Equal(t, "/path/on/disk/", path)
				return tc.tarList, nil
			})
			result, version, err := identify.Identify(testcontext.GetTestContext(t), "/path/on/disk/", stubDirEntry{
				name: tc.filename,
			})
			require.NoError(t, err)
			assert.Equal(t, tc.result, result)
			var expectedVersion string
			if tc.version == "" {
				expectedVersion = crawl.UnknownVersion
			} else {
				expectedVersion = tc.version
			}
			assert.Equal(t, expectedVersion, version)
		})
	}
}

type stubDirEntry struct {
	name string
}

func (s stubDirEntry) Name() string {
	return s.name
}

func (s stubDirEntry) IsDir() bool {
	panic("not required")
}

func (s stubDirEntry) Type() fs.FileMode {
	panic("not required")
}

func (s stubDirEntry) Info() (fs.FileInfo, error) {
	panic("not required")
}

func panicOnList(context.Context, string) ([]string, error) {
	panic("should not have been called")
}
