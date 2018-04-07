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
  "os"
  "log"
  "gopkg.in/ini.v1"
)

type global_config struct {
  email       string
  smtp_from   string
  smtp_server string
  smtp_port   int
  log_level   int
}

/*
 * loads configuration from file
 */
func load_config(fname string) (gconf global_config, watches []watchPath) {
  cfg, err := ini.ShadowLoad(fname)
  if err != nil {
    log.Fatal(fmt.Sprintf("Fail to read file: %v\n", err))
  }

  // process global config
  myhostname,_ := os.Hostname()
  gconf.email       = cfg.Section(ini.DEFAULT_SECTION).Key("email").String()
  gconf.smtp_from   = cfg.Section(ini.DEFAULT_SECTION).Key("smtp_from").MustString(fmt.Sprintf("fscanary@%s", myhostname))
  gconf.smtp_server = cfg.Section(ini.DEFAULT_SECTION).Key("smtp_server").String()
  gconf.smtp_port   = cfg.Section(ini.DEFAULT_SECTION).Key("smtp_port").MustInt(25)
  gconf.log_level   = cfg.Section(ini.DEFAULT_SECTION).Key("logging").MustInt(1)

  // validate config
  if gconf.log_level < 0 || gconf.log_level > 9 {
    log.Fatal("Configuration option 'logging' must be between 0 and 9")
  }

  // process watch sections
  section_names := cfg.SectionStrings()
  for _, sec_name := range section_names {
    if sec_name == "DEFAULT" {
      continue
    }
    if cfg.Section(sec_name).Key("enabled").MustBool(true) == false {
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


