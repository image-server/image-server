package file_garbage_collector

import (
	"log"
	"os"
	"time"

	"github.com/image-server/image-server/core"
	"io"
	"path/filepath"
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
					log.Printf("[tickID: %v] Started\n", tickTime)
					stepNum := 0
					pStepNum := &stepNum
					var pathsToDelete []string
					pPathsToDelete := &pathsToDelete
					filepath.Walk(absolutePath, func(path string, info os.FileInfo, err error) error {
						*pStepNum = *pStepNum + 1
						if err != nil {
							log.Printf("[tickID: %v] Error walking path [%s] step [%v]\n", tickTime, path, *pStepNum)
							return err
						}
						if info.IsDir() {
							empty, err := IsDirectoryEmpty(path)
							if err != nil {
								log.Printf("[tickID: %v] Error determining if directory is empty [%s] step [%v]\n", tickTime, path, *pStepNum)
								return err
							}
							if empty && absolutePath != path {
								age := tickTime.Sub(info.ModTime())
								log.Printf("[tickID: %v] Marking deletion of directory [%s] size [%d] modTime [%s] age [%s] step [%v]\n", tickTime, path, info.Size(), info.ModTime(), age, *pStepNum)
								*pPathsToDelete = append(*pPathsToDelete, path)
							}
						} else {
							age := tickTime.Sub(info.ModTime())
							if age > sc.MaxFileAge {
								log.Printf("[tickID: %v] Marking deletion of file [%s] size [%d] modTime [%s] age [%s] step [%v]\n", tickTime, path, info.Size(), info.ModTime(), age, *pStepNum)
								*pPathsToDelete = append(*pPathsToDelete, path)
							}
						}
						return nil
					})
					log.Printf("[tickID: %v] Preparing to delete [%v] items\n", tickTime, len(pathsToDelete))
					for _, path := range pathsToDelete {
						err = os.Remove(path)
						if err != nil {
							log.Printf("[tickID: %v] Error deleting [%s]\n", tickTime, path)
						}
					}

					log.Printf("[tickID: %v] Finished in [%v] steps\n", tickTime, stepNum)
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
