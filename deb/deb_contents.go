// +build !opt !cgo

package deb

import (
	"io"
	"compress/gzip"
	"fmt"
	"compress/bzip2"
	"archive/tar"
	"strings"
	"os"
	"bufio"

	"github.com/h2non/filetype/matchers"
	"github.com/mkrautz/goar"
	"github.com/smira/go-xz"
	"github.com/smira/lzma"
)

// GetContentsFromDeb returns list of files installed by .deb package
func GetContentsFromDeb(packageFile string) ([]string, error) {
	file, err := os.Open(packageFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	library := ar.NewReader(file)
	for {
		header, err := library.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("unable to find data.tar.* part in %s", packageFile)
		}
		if err != nil {
			return nil, fmt.Errorf("unable to read .deb archive from %s: %s", packageFile, err)
		}

		if strings.HasPrefix(header.Name, "data.tar") {
			bufReader := bufio.NewReader(library)
			signature, err := bufReader.Peek(270)

			var isTar bool
			if err == nil {
				isTar = matchers.Tar(signature)
			}

			var tarInput io.Reader

			switch header.Name {
			case "data.tar":
				tarInput = bufReader
			case "data.tar.gz":
				if isTar {
					tarInput = bufReader
				} else {
					ungzip, err := gzip.NewReader(bufReader)
					if err != nil {
						return nil, fmt.Errorf("unable to ungzip data.tar.gz from %s: %s", packageFile, err)
					}
					defer ungzip.Close()
					tarInput = ungzip
				}
			case "data.tar.bz2":
				tarInput = bzip2.NewReader(bufReader)
			case "data.tar.xz":
				unxz, err := xz.NewReader(bufReader)
				if err != nil {
					return nil, fmt.Errorf("unable to unxz data.tar.xz from %s: %s", packageFile, err)
				}
				defer unxz.Close()
				tarInput = unxz
			case "data.tar.lzma":
				unlzma := lzma.NewReader(bufReader)
				defer unlzma.Close()
				tarInput = unlzma
			default:
				return nil, fmt.Errorf("unsupported tar compression in %s: %s", packageFile, header.Name)
			}

			untar := tar.NewReader(tarInput)
			var results []string
			for {
				tarHeader, err := untar.Next()
				if err == io.EOF {
					return results, nil
				}
				if err != nil {
					return nil, fmt.Errorf("unable to read .tar archive from %s: %s", packageFile, err)
				}

				if tarHeader.Typeflag == tar.TypeDir {
					continue
				}

				if strings.HasPrefix(tarHeader.Name, "./") {
					tarHeader.Name = tarHeader.Name[2:]
				}
				results = append(results, tarHeader.Name)
			}
		}
	}
}
