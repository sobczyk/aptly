// +build opt,cgo

package deb

/*
#cgo CFLAGS: -std=c99
#cgo LDFLAGS: -larchive
#include <stdlib.h>
#include <time.h>
#include <archive.h>
#include <archive_entry.h>
#include <fnmatch.h>
#include <stdbool.h>

struct a
{
	struct archive *deb, *data;
	struct archive_entry *deb_entry, *data_entry;
	void *buffer[10240];
	const char *err;
};

ssize_t read_a(struct archive *a, void *data, const void **buff)
{
	struct a *internal = (struct a*)data;
	*buff = internal->buffer;
	int read = archive_read_data(internal->deb, internal->buffer, 10240);
	if (read < 0)
	{
		return -1;
	}
	return read;
}

struct a* open_archive(char *file, bool *error)
{
	struct a *internal = malloc(sizeof(struct a));
	*error = false;
	internal->deb = archive_read_new();
	archive_read_support_format_ar(internal->deb);
	archive_read_open_filename(internal->deb, file, 10240);
	free(file);
	if (!internal->deb)
	{
		return 0;
	}
	while (archive_read_next_header(internal->deb, &internal->deb_entry) == ARCHIVE_OK)
	{
		const char* name = archive_entry_pathname(internal->deb_entry);
		if (!fnmatch("data.*", name, 0))
		{
			internal->data = archive_read_new();
			archive_read_support_format_tar(internal->data);
			archive_read_support_filter_all(internal->data);
			int r = archive_read_open(internal->data, internal, 0, read_a, 0);
			if (r != ARCHIVE_OK)
			{
				internal->err = archive_error_string(internal->data);
				*error = true;
				return internal;
			}
			return internal;
		}
	}
	return 0;
}

const char* read_content(struct a *internal, bool *error)
{
	int r;
	*error = false;
	for (;;)
	{
		r = archive_read_next_header(internal->data, &internal->data_entry);
		if (r == ARCHIVE_EOF)
		{
			return 0;
		}
		if (r != ARCHIVE_OK)
		{
			*error = true;
			internal->err = archive_error_string(internal->data);
			return 0;
		}
		if (archive_entry_filetype(internal->data_entry) == AE_IFDIR)
			continue;
		return archive_entry_pathname(internal->data_entry);
	}
}

int close_archive(struct a *internal)
{
	if (internal->data)
	{
		archive_read_close(internal->data);
		archive_read_free(internal->data);
	}
	if (internal->deb)
	{
		archive_read_close(internal->deb);
		archive_read_free(internal->deb);
	}
	free(internal);
	return 0;
}

const char *get_error(struct a *internal)
{
	return internal->err;
}

*/
import "C"

import (
"fmt"
"strings"
)

// GetContentsFromDeb returns list of files installed by .deb package
func GetContentsFromDeb(packageFile string) ([]string, error) {
	err := C.bool(true);
	f := C.open_archive(C.CString(packageFile), &err)
	if f == nil {
		return nil, fmt.Errorf("Error initializing libarchive")
	}
	defer C.close_archive(f)
	if err {
		return nil, fmt.Errorf("Error initializing libarchive")
	}

	var results []string
	for {
		arg := C.GoString(C.read_content(f, &err))
		if err {
			return nil, fmt.Errorf("Error during libarchive decompress: %s", C.GoString(C.get_error(f)))
		}
		if len(arg) == 0 {
			return results, nil
		}
		if strings.HasPrefix(arg, "./") {
			arg = arg[2:]
		}
		results = append(results, arg)

	}
	return results, nil
}