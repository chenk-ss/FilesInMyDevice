package main

import (
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Todo struct {
	Title string
	Done  bool
}

type PageData struct {
	PageTitle string
	Todos     []Todo
}

var base_path string
var base_port = "7005"
var base_download_port string
var base_domain string
var base_url string
var base_download_url string

type QueryFileInfoParam struct {
	Path string `form:"path" json:"path"`
}

func Query(c *gin.Context) {
	var param QueryFileInfoParam
	if err := c.ShouldBind(&param); err != nil {
		c.AbortWithStatusJSON(http.StatusOK, gin.H{
			"code":    400,
			"message": "err",
			"data":    err,
		})
		return
	}
	path := param.Path
	if len(path) == 0 {
		path += "/"
	}
	if len(path) > 0 {
		if path[len(path)-1] != '/' {
			path += "/"
		}
		paths := strings.Split(path, "/")
		if len(paths) > 2 && paths[len(paths)-2] == ".." {
			path = strings.Join(paths[:len(paths)-3], "/")
			path += "/"
		}
	}
	files := QueryFiles(path)
	c.HTML(http.StatusOK, "index.html", gin.H{
		"path":  path,
		"back":  base_url + "?path=" + path + "..",
		"files": files,
	})
}

type File struct {
	Name    string
	Type    int
	Address string
	Size    string
}

func QueryFiles(path string) []File {
	files, err := os.ReadDir(base_path + path)
	if err != nil {
		log.Panicln(err)
		return nil
	}
	res := []File{}
	for i := range files {
		file := files[i]
		if file.Name()[0] == '.' {
			continue
		}
		nFile := File{}
		if file.IsDir() {
			nFile.Name = file.Name() + "/"
			nFile.Type = 0
			nFile.Address = base_url + "?path=" + path + file.Name() + "/"
		} else {
			nFile.Name = file.Name()
			nFile.Type = 1
			nFile.Address = base_download_url + path + file.Name()
			info, _ := file.Info()
			size := info.Size()
			switch {
			case size >= 1<<30:
				nFile.Size = fmt.Sprintf("[%.2fG]", float64(size)/float64(1<<30))
			case size >= 1<<20:
				nFile.Size = fmt.Sprintf("[%.2fMB]", float64(size)/float64(1<<20))
			case size >= 1<<10:
				nFile.Size = fmt.Sprintf("[%.2fKB]", float64(size)/float64(1<<10))
			default:
				nFile.Size = fmt.Sprintf("[%.fB]", float64(size))
			}
		}
		res = append(res, nFile)
	}
	sort.Slice(res, func(i, j int) bool {
		mi, mj := res[i], res[j]
		switch {
		case mi.Type == mj.Type:
			return sortName(mi.Name) < sortName(mj.Name)
		default:
			return mi.Type < mj.Type
		}
	})
	return res
}

func sortName(filename string) string {
	ext := filepath.Ext(filename)
	name := filename[:len(filename)-len(ext)]
	// split numeric suffix
	i := len(name) - 1
	for ; i >= 0; i-- {
		if '0' > name[i] || name[i] > '9' {
			break
		}
	}
	i++
	// string numeric suffix to uint64 bytes
	// empty string is zero, so integers are plus one
	b64 := make([]byte, 64/8)
	s64 := name[i:]
	if len(s64) > 0 {
		u64, err := strconv.ParseUint(s64, 10, 64)
		if err == nil {
			binary.BigEndian.PutUint64(b64, u64+1)
		}
	}
	// prefix + numeric-suffix + ext
	return name[:i] + string(b64) + ext
}

func args() {
	flag.StringVar(&base_port, "P", "", "Port is nil")
	flag.StringVar(&base_download_port, "DP", "", "Download Port is nil")
	flag.StringVar(&base_path, "PATH", "", "Base Path is nil")
	flag.StringVar(&base_domain, "DOMAIN", "", "Base Domain is nil")
	flag.Parse()
	base_url = base_domain + ":" + base_port
	base_download_url = base_domain + ":" + base_download_port
	log.Printf("Params:\n%s\n%s\n%s\n%s\n%s\n%s", base_port, base_download_port, base_path, base_domain, base_url, base_download_url)
}

func main() {
	args()
	r := gin.Default()
	r.LoadHTMLFiles("index.html")
	r.GET("/", Query)

	srv := &http.Server{
		Addr:    ":" + base_port,
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Errorf("start listen : %s\n", err)
		}
	}()
	// if have two quit signal , this signal will priority capture ,also can graceful shutdown
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-quit
	logrus.Infof("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logrus.Error("Server Shutdown:", err)
	}
	logrus.Infof("Server exiting")
	os.Exit(0)
}
