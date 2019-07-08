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
  "log"
  "os"
  "path"
  "time"
  "github.com/rjeczalik/notify"
  "github.com/gobwas/glob"
  "gopkg.in/gomail.v2"
)

/*
 * STRUCTS
 */
type watchPath struct {
  title   string    // name of the watch
  path    []string  // path(s) to watch
  pattern []string  // pattern(s) to match
  exclude []string  // pattern(s) to exclude
  notify  bool      // should we notify for files that match?
  qtine   bool      // should we quarantine files that match?
  qdest   string    // destination to quarantine files to
}
type notification struct {
  dest    string  // recipient of notification
  subject string  // subject of notificaton
  msg     string  // notification text
  ts      int64   // timestamp notification was created
}

/*
 * MAIN
 */
var gconf global_config
var watches []watchPath
var version string  // set at build time by -ldflags
var pending_notifs []notification

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

  logmsg(1, "Creating watches. This may take a long time for large directory trees. Please wait for Ready message.")
  for _,watch := range watches {
    logmsg(2, fmt.Sprintln("Adding watch: ", watch.title))
    for _,path := range watch.path {
      logmsg(2, fmt.Sprintln("  Path: ", path))
      addWatch(chNotify, path)
    }
  }
  defer notify.Stop(chNotify)

  // Block until an event is received.
  logmsg(0,"Ready")
  for {
    select {
    case ei := <-chNotify:
      logmsg(5, fmt.Sprintln("Got event:", ei))
      handle_event(ei)
    case <-chQuit:
      logmsg(1, "QUIT")
      os.Exit(0)
    }
    // async sending of pending_notifs
    go send_pending_notifs()
  }
}

/******************************************************************************
 * FUNCTIONS
 *****************************************************************************/
func addWatch(ch chan notify.EventInfo, path string) {
  if err := notify.Watch(path + string(os.PathSeparator) + "...", ch, notify.Create|notify.Write|notify.Rename); err != nil {
    log.Fatal(err)
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
    logmsg(5, fmt.Sprintf("Skipping Event (is directory): %s", ei.Path()))
    return
  }

  // strip leading path fom ei.Path() so we can compare patterns against the
  // filename only rather than the full path
  fname := path.Base(ei.Path())
  for _,watch := range watches {
    // check for matching patterns
    compare_paths:
    for _,pattern := range watch.pattern {
      logmsg(2, fmt.Sprintf("Comparing '%s' to '%s'", fname, pattern))
      var g_path glob.Glob
      g_path = glob.MustCompile(pattern)
      if g_path.Match(fname) {
        for _,exclude := range watch.exclude {
          var g_exclude glob.Glob
          g_exclude = glob.MustCompile(exclude)
          if g_exclude.Match(fname) {
            logmsg(5, fmt.Sprintf("File '%s' match exclude pattern '%s'; skipping", fname, exclude))
            continue compare_paths
          }
        }
        logmsg(4, "Match found")
        if watch.notify {
          // generate a notification
          myhostname,_ := os.Hostname()
          logmsg(1, fmt.Sprintf("Generating notification for %s event on file '%s'",
            event_verb, ei.Path()))
          var n notification
          n.dest = gconf.email
          n.subject = fmt.Sprintf("File Matching '%s'", watch.title)
          n.msg = fmt.Sprintf("The following file matching pattern '%s' was saved on host %s\n\n%s",
                       pattern, myhostname, ei.Path())
          t := time.Now().UTC()
          n.ts = t.Unix()
          pending_notifs = append(pending_notifs, n)
        }
        if watch.qtine {
          // quarantine the file
          logmsg(1, fmt.Sprintf("Quarantining '%s' to '%s'",
            ei.Path(), watch.qdest))
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
  logmsg(3, fmt.Sprintf("Finished examining '%s'", ei.Path()))
}

func send_pending_notifs() {
  curr_notifs := pending_notifs  // copy pending_notifs to a new slice
  pending_notifs = nil          // erase the global pending_notifs slice
  for _, n := range curr_notifs {
    m := gomail.NewMessage()
    m.SetHeader("From", gconf.smtp_from)
    m.SetHeader("To", n.dest)
    m.SetHeader("Subject", n.subject)
    m.SetBody("text/plain", n.msg)

    logmsg(1, fmt.Sprintf("Sending notification '%s' to '%s'", n.subject, n.dest))
    d := gomail.Dialer{Host: gconf.smtp_server, Port: gconf.smtp_port}
    if err := d.DialAndSend(m); err != nil { panic(err) }
  }
}
