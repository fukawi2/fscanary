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
  "fmt"
  "flag"
  "os"
  "io"
  "path/filepath"
  "os/signal"
  "log"
  "gopkg.in/ini.v1"
  "github.com/rjeczalik/notify"
  "github.com/gobwas/glob"
  "gopkg.in/gomail.v2"
)

/*
 * STRUCTS
 */
type global_config struct {
  email       string
  smtp_from   string
  smtp_server string
  smtp_port   int
}

type watchPath struct {
  title   string    // name of the watch
  path    []string  // path(s) to watch
  pattern []string  // pattern(s) to match
  notify  bool      // should we notify for files that match?
  qtine   bool      // should we quarantine files that match?
  qdest   string    // destination to quarantine files to
}

/*
 * MAIN
 */
var gconf global_config
var watches []watchPath
var version string  // set at build time by -ldflags
func main() {
  // process command line arguments
  var conf_fname string
  var show_version bool
  flag.StringVar(&conf_fname, "config", default_conf_fname, "configuration file to load")
  flag.BoolVar(&show_version, "version", false, "show version information")
  flag.Parse()

  if show_version {
    fmt.Println("fscanary", version)
    fmt.Println("Copyright 2018 Phillip Smith. Licensed under MIT license")
    os.Exit(0)
  }

  // setup signal catching
  sigs := make(chan os.Signal, 1)
  // catch all signals since not explicitly listing
  signal.Notify(sigs)
  // method invoked upon seeing signal
  go func() {
    s := <-sigs
    log.Printf("RECEIVED SIGNAL: %s", s)
    os.Exit(1)
  }()

  // load configuration file
  gconf, watches = load_config(conf_fname)

  // Validate configuration
  for _,watch := range watches {
    if watch.qtine {
      // must have a valid destination to quarantine to
      if len(watch.qdest) == 0 {
        // dest dir no configured!
        log.Fatal(fmt.Sprintf(
          "'%s' has quarantine enabled but no destination directory configured",
          watch.title))
      }
      if x,_ := path_is_dir(watch.qdest); x != true {
        // dest dir does not exist
        log.Fatal(fmt.Sprintf(
          "'%s' quarantine destination '%s' does not exist",
          watch.title, watch.qdest))
      }
    }
  }

  /*
   * Make the channel buffered to ensure no event is dropped. Notify will drop
   * an event if the receiver is not able to keep up the sending pace.
   */
  chNotify  := make(chan notify.EventInfo, 16)
  chQuit    := make(chan int)

  for _,watch := range watches {
    fmt.Println("Adding watch: ", watch.title)
    for _,path := range watch.path {
      fmt.Println("  Path: ", path)
      if err := notify.Watch(path, chNotify, notify.Create|notify.Write|notify.Rename); err != nil {
          log.Fatal(err)
      }
    }
  }
  defer notify.Stop(chNotify)

  // Block until an event is received.
  for {
    select {
    case ei := <-chNotify:
      //log.Println("Got event:", ei)
      handle_event(ei)
    case <-chQuit:
      log.Println("QUIT")
      os.Exit(0)
    }
  }
}

/*
 * FUNCTIONS
 */
func load_config(fname string) (gconf global_config, watches []watchPath) {
  cfg, err := ini.ShadowLoad(fname)
  if err != nil {
    fmt.Printf("Fail to read file: %v\n", err)
    os.Exit(1)
  }

  // process global config
  myhostname,_ := os.Hostname()
  gconf.email       = cfg.Section(ini.DEFAULT_SECTION).Key("email").String()
  gconf.smtp_from   = cfg.Section(ini.DEFAULT_SECTION).Key("smtp_from").MustString(fmt.Sprintf("fscanary@%s", myhostname))
  gconf.smtp_server = cfg.Section(ini.DEFAULT_SECTION).Key("smtp_server").String()
  gconf.smtp_port   = cfg.Section(ini.DEFAULT_SECTION).Key("smtp_port").MustInt(25)

  // process watch sections
  section_names := cfg.SectionStrings()
  for _, sec_name := range section_names {
    if sec_name == "DEFAULT" {
      continue
    }
    var watch watchPath
    watch.title   = sec_name
    watch.path    = cfg.Section(sec_name).Key("path").ValueWithShadows()
    watch.pattern = cfg.Section(sec_name).Key("pattern").ValueWithShadows()
    watch.notify  = cfg.Section(sec_name).Key("notify").MustBool(true)
    watch.qtine   = cfg.Section(sec_name).Key("quarantine").MustBool(false)
    watch.qdest   = cfg.Section(sec_name).Key("dest").String()

    watches = append(watches, watch)
  }

  return gconf, watches
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
 * examines a notify.EventInfo and compares it against our
 * watch configuration for matching files.
 */
func handle_event(ei notify.EventInfo) {
  var event_verb string
  switch ei.Event() {
    case notify.Create: event_verb = "Created"
    case notify.Write:  event_verb = "Modified"
    case notify.Rename: event_verb = "Renamed"
    default:            event_verb = "Unknown"
  }
  _ = event_verb

  // abort if file is actually a directory
  if x,_ := path_is_dir(ei.Path()); x == true {
    fmt.Println("Ignoring directory:", ei.Path())
    return
  }

  for _,watch := range watches {
    // check for matching patterns
    for _,pattern := range watch.pattern {
      var g glob.Glob
      g = glob.MustCompile(pattern)
      if g.Match(ei.Path()) {
        if watch.notify {
          // send notification
          fmt.Println(ei.Path(), "triggered notification to", gconf.email)
          myhostname,_ := os.Hostname()
          m := gomail.NewMessage()
          m.SetHeader("From", gconf.smtp_from)
          m.SetHeader("To", gconf.email)
          m.SetHeader("Subject",
            fmt.Sprintf("File Matching '%s'", watch.title))
          m.SetBody("text/plain",
            fmt.Sprintf("The following file matching pattern '%s' was saved on host %s\n\n%s",
            pattern, myhostname, ei.Path()))

          d := gomail.Dialer{Host: gconf.smtp_server, Port: gconf.smtp_port}
          if err := d.DialAndSend(m); err != nil {
            panic(err)
          }
        }
        if watch.qtine {
          // quarantine the file
          fmt.Println(ei.Path(), "being quarantined to", watch.qdest)
          err := moveFile(ei.Path(), watch.qdest + ei.Path())
          if err != nil {
            fmt.Println(err)
            return
          }
        }
        break
      }
    }
  }
}

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
  if _, err = io.Copy(fdst, fsrc); err != nil {
    return err
  }

  if err == nil { err = fsrc.Close() }
  if err == nil { err = fdst.Close() }
  if err == nil { err = os.Chmod(dst, fi.Mode()) }
  if err == nil { err = os.Chtimes(dst, fi.ModTime(), fi.ModTime()) }
  if err == nil { err = os.Remove(src) }

  return nil
}
