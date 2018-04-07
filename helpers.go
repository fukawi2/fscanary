/*
MIT License

Copyright (c) 2018 Phillip Smith

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package main

import (
  "os"
  "fmt"
  "io"
  "path/filepath"
  "log"
  "strings"
)

/*
 * handles displaying of log messages
 */
func logmsg(lvl int, msg string) error {
  if (lvl > 9) {
    return nil
  }
  log.Println(strings.TrimSuffix(msg, "\n"))
  return nil
}

/*
 * tests a path to determine if it is a directory or other file
 */
func path_is_dir(path string) (bool, error) {
  fi, err := os.Stat(path);
  switch {
  case err != nil:
    // handle the error and return
    return false, nil
  case fi.IsDir():
    // it's a directory
    return true, nil
  default:
    // it's not a directory
    return false, nil
  }
}

/*
 * moves a file on disk
 */
func moveFile(src string, dst string) error {
  // make sure the file exists by stat'ing it
  fi, err := os.Stat(src)
  if err != nil { return err }

  // create any leading directory structure for the destination
  if err = os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
    fmt.Printf("MkdirAll(%v)\n", filepath.Dir(dst))
    return err
  }

  // open the source and destination files
  fsrc, err := os.Open(src)
  if err != nil { return err }

  fdst, err := os.Create(dst)
  if err != nil { return err }

  // copy from old to new
  if _, err = io.Copy(fdst, fsrc); err != nil { return err }

  if err == nil { err = fsrc.Close() }
  if err == nil { err = fdst.Close() }
  if err == nil { err = os.Chmod(dst, fi.Mode()) }
  if err == nil { err = os.Chtimes(dst, fi.ModTime(), fi.ModTime()) }
  if err == nil { err = os.Remove(src) }

  return nil
}
