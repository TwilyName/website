package fileindex

import (
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/TwilyName/website/config"
	"github.com/shurcooL/httpfs/filter"
	"github.com/shurcooL/httpfs/vfsutil"
)

func byteCountIEC(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func (h *handler) isHiddenPath(p string, clientAddr net.IP) bool {
	hidden := config.Get().Handlers.FileIndex.Hide

	for _, hiddenEntry := range hidden {
		cond := regexp.MustCompile(hiddenEntry.Regex)
		if cond.Match([]byte(p)) && !config.IsAllowedByACL(clientAddr, hiddenEntry.Exclude) {
			return true
		}
	}

	return false
}

type DirContentCallback func(relativepath string, fi fs.FileInfo, err error) error

func (h *handler) getDirContent(
	basepath string,
	addr net.IP,
	recursive bool,
	searchParams searchParams,
	callback DirContentCallback,
) error {
	if h.isHiddenPath(basepath, addr) {
		return errors.New("Content not found")
	}

	newroot := filter.Skip(h.root, func(path string, fi os.FileInfo) bool {
		if fi.IsDir() {
			path = path + "/"
		}
		return h.isHiddenPath(path, addr)
	})

	if recursive {
		onetimeskip := false
		err := vfsutil.Walk(newroot, basepath, func(path string, fi fs.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !onetimeskip { // skip the basepath directory itself
				onetimeskip = true
				return err
			}

			name := fi.Name()
			nameMatches, _ := h.nameMatchesSearchParams(name, searchParams)
			if !nameMatches {
				return err
			}

			relativepath := path[len(basepath):]
			if len(relativepath) > 0 {
				relativepath = relativepath[:len(relativepath)-len(fi.Name())]
			}
			return callback(relativepath, fi, err)
		})
		return err
	}

	filelist, err := vfsutil.ReadDir(newroot, basepath)
	for _, fi := range filelist {
		err = callback("", fi, nil)
		if err != nil {
			return err
		}
	}
	return err
}

type fileEntry struct {
	Name    string
	Size    string
	Date    string
	IsDir   bool
	RawSize int64
	RawDate time.Time
}

func (h *handler) prepareFileList(path string, addr net.IP, search searchParams, sorting sortParams) ([]fileEntry, error) {
	result := make([]fileEntry, 0)
	hasQuery := len(search.FindQuery) > 0

	err := h.getDirContent(path, addr, hasQuery, search, func(relativepath string, fi fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relativepath = strings.TrimLeft(relativepath, "/")
		name := fi.Name()
		fpath := fmt.Sprintf("%s%s", relativepath, name)

		fi, err = h.readSymlink(filepath.Join(path, relativepath), fi)
		if err != nil {
			return err
		}

		if fi.IsDir() {
			name = name + "/"
			fpath = fpath + "/"
		}

		check, err := h.nameMatchesSearchParams(name, search)
		if err != nil || !check {
			return err
		}

		entryName := name
		if hasQuery {
			entryName = fpath
		}

		result = append(result, fileEntry{
			IsDir:   fi.IsDir(),
			Name:    entryName,
			RawSize: fi.Size(),
			RawDate: fi.ModTime().UTC(),
			Date:    fi.ModTime().UTC().Format("2006-01-02 15:04:05"),
			Size:    byteCountIEC(fi.Size()),
		})

		return err
	})
	if err != nil {
		return result, err
	}

	if len(result) != 0 {
		sort.Slice(result, func(i, j int) bool {
			if result[i].IsDir != result[j].IsDir {
				return result[i].IsDir
			}
			return sortByParams(result[i], result[j], sorting)
		})
	}

	return result, err
}

func (h *handler) readSymlink(path string, fi fs.FileInfo) (os.FileInfo, error) {
	if fi.Mode()&fs.ModeSymlink == 0 {
		return fi, nil
	}

	fullpath := filepath.Join(config.Get().Paths.Base, path, fi.Name())
	realpath, err := filepath.EvalSymlinks(fullpath)
	if err != nil {
		return fi, err
	}

	return os.Stat(realpath)
}

func sortByParams(entry1, entry2 fileEntry, params sortParams) bool {
	var diff int64 = -1

	if params.Field == SortByName {
		name1 := strings.ToLower(entry1.Name)
		name2 := strings.ToLower(entry2.Name)
		diff = int64(strings.Compare(name1, name2))
	}

	if params.Field == SortBySize {
		diff = entry1.RawSize - entry2.RawSize
	}

	if params.Field == SortByDate {
		diff = int64(entry1.RawDate.Compare(entry2.RawDate))
	}

	if params.IsDesc {
		return diff > 0
	}
	return diff < 0
}
