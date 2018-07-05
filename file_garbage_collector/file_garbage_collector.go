package file_garbage_collector

import (
	"log"
	"os"
	"time"

	"github.com/image-server/image-server/core"
	"path/filepath"
	"io"
)

func Start(sc *core.ServerConfiguration) {
	go func() {
		absolutePath, err := filepath.Abs(sc.LocalBasePath)
		if err != nil {
			log.Printf("Error Starting File Cleaner - Unable to create absolute path [%s]", sc.LocalBasePath)
		} else {
			var stat, _ = os.Stat(absolutePath)
			if absolutePath != "" && stat.IsDir() {
				log.Printf("Starting File Cleaner on path [%s]", absolutePath)
				filepath.Walk(absolutePath, func(path string, info os.FileInfo, err error) error {
					if !info.IsDir() {
						age := time.Now().Sub(info.ModTime())
						log.Printf("Workspace Contains at Startup file [%s] size [%d] modTime [%s] age [%s]\n", path, info.Size(), info.ModTime(), age)
					}
					return nil
				})
				for range sc.CleanUpTicker.C {
					tickTime := time.Now()
					log.Printf("Starting [tickID: %v]\n", tickTime)
					stepNum := 0
					filepath.Walk(absolutePath, func(path string, info os.FileInfo, err error) error {
						stepNum = stepNum + 1
						if err != nil {
							return err
						}
						if info.IsDir() {
							empty, err := IsDirectoryEmpty(path)
							if err != nil {
								return err
							}
							if empty {
								age := tickTime.Sub(info.ModTime())
								log.Printf("[tickID: %v] Deleting directory [%s] size [%d] modTime [%s] age [%s]\n", tickTime, path, info.Size(), info.ModTime(), age)
								os.Remove(path)
							}
						} else {
							age := tickTime.Sub(info.ModTime())
							if age > sc.MaxFileAge {
								log.Printf("[tickID: %v] Deleting file [%s] size [%d] modTime [%s] age [%s]\n", tickTime, path, info.Size(), info.ModTime(), age)
								var err = os.Remove(path)
								if err != nil {
									log.Printf("[tickID: %v] Error deleting file [%s]\n", tickTime, path)
								}
							}
						}
						return nil
					})
					log.Printf("Finished [tickID: %v] in [%v] steps\n", tickTime, stepNum)
				}
			} else {
				log.Printf("Error Starting File Cleaner - Invalid walk path [%s]", absolutePath)
			}
		}
	}()
}

func IsDirectoryEmpty(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1) // Or f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err // Either not empty or error, suits both cases
}