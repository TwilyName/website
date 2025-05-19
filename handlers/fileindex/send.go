package fileindex

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"path/filepath"

	"github.com/TwilyName/website/handlers/util"
	"github.com/klauspost/compress/zstd"
)

func (h *handler) sendFile(writer http.ResponseWriter, req *http.Request, params searchParams) (bool, error) {
	file, err := h.root.Open(req.URL.Path)
	if err != nil {
		return false, err
	}

	stat, err := file.Stat()
	if err != nil {
		return false, err
	}

	if stat.Mode().IsDir() {
		if uploader, ok := h.uploaders[req.URL.Query().Get("download")]; ok {
			return uploader(writer, req, params)
		}
	}

	if !stat.Mode().IsRegular() {
		return false, nil
	}

	http.ServeContent(writer, req, stat.Name(), stat.ModTime(), file)
	return true, nil
}

func (h *handler) prepareTar(w io.WriteCloser, dir string, clientAddr net.IP, params searchParams) error {
	defer w.Close()

	tw := tar.NewWriter(w)
	defer tw.Close()

	_, dirname := filepath.Split(filepath.Dir(dir))

	return h.getDirContent(dir, clientAddr, true, params, func(relativepath string, fi fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		th, err := tar.FileInfoHeader(fi, "") // TODO: better link argument?
		if err != nil {
			return err
		}

		// hide uid:gid, set them to nobody
		th.Uid = 65534
		th.Gid = 65534
		th.Uname = "nobody"
		th.Gname = "nobody"
		th.Name = fmt.Sprintf("%s/%s%s", dirname, relativepath, fi.Name())

		if err = tw.WriteHeader(th); err != nil {
			return err
		}
		if fi.IsDir() {
			return err
		}

		path := fmt.Sprintf("%s%s%s", dir, relativepath, fi.Name())
		fh, err := h.root.Open(path)
		if err != nil {
			return err
		}
		defer fh.Close()

		_, err = io.Copy(tw, fh)
		return err
	})
}

func (h *handler) uploadTar(w http.ResponseWriter, r *http.Request, params searchParams) (bool, error) {
	dir := r.URL.Path[1:]
	filename := fmt.Sprintf("%s.tar", filepath.Base(r.URL.Path))

	w.Header().Add("Content-Type", "application/x-tar")
	w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	bufr, bufw := io.Pipe()
	defer bufr.Close()

	go h.prepareTar(bufw, dir, util.GetRemoteAddr(r), params)
	written, err := io.Copy(w, bufr)
	return written > 0, err
}

func (h *handler) uploadGz(w http.ResponseWriter, r *http.Request, params searchParams) (bool, error) {
	dir := r.URL.Path[1:]
	filename := fmt.Sprintf("%s.tar.gz", filepath.Base(r.URL.Path))

	w.Header().Add("Content-Type", "application/gzip")
	w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	bufr, bufw := io.Pipe()
	defer bufr.Close()

	compressor := gzip.NewWriter(w)
	defer compressor.Close()

	go h.prepareTar(bufw, dir, util.GetRemoteAddr(r), params)
	written, err := io.Copy(compressor, bufr)
	return written > 0, err
}

func (h *handler) uploadZst(w http.ResponseWriter, r *http.Request, params searchParams) (bool, error) {
	dir := r.URL.Path[1:]
	filename := fmt.Sprintf("%s.tar.zst", filepath.Base(r.URL.Path))

	w.Header().Add("Content-Type", "application/zstd")
	w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	bufr, bufw := io.Pipe()
	defer bufr.Close()

	compressor, err := zstd.NewWriter(w)
	if err != nil {
		return false, err
	}
	defer compressor.Close()

	go h.prepareTar(bufw, dir, util.GetRemoteAddr(r), params)
	written, err := io.Copy(compressor, bufr)
	return written > 0, err
}
