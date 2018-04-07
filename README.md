# fscanary

Monitor and report or action file systems for specific files.

## Description

fscanary aims to be a replacement for some features of Microsoft's File Server
Resource Manager (FSRM), particularly in detecting creation of specific files
and notifying the system admin of those files.

For example, it is likely to be desirable to have an alert when a user saves
and executable file to the company file server.

## Getting Started

Deploy `fscanary` to your preferred bin location (eg, `/usr/local/bin/fscanary`)

### Configuration

The default location of the config file is OS-specific and is shown in the
output of `fscnary -help`. It can be overridden with the `-config` command line
argument -config. For example:

```
fscanary -config /root/fscanary.conf
```

The configuration file is INI-style and has a global section for general
configuration, then one or more sections to set up watches.

Example configuration file:

```
email = admin@example.com
smtp_server = smarthost
smtp_from = fscanary@example.com

[executables]
enabled = yes
path = /home
notify = yes
quarantine = yes
dest = /quarantined
pattern = *.exe
pattern = *.bat
pattern = *.js
pattern = *.bin
pattern = *.cmd
pattern = *.cpl
pattern = *.dll
pattern = *.lnk
pattern = *.ps1
pattern = *.scr
```

## Contributing

Pull requests welcomed! Please keep the following points in mind:

* One feature change/bug fix per pull request.
  * Multiple minor changes such as spelling may be in a single PR.
* Documentation must be updated with the PR where the change warrants
  documentation changes.
* Check the TODO.md file for specific tasks that are outstanding.

## Authors

* **Phillip Smith** - [fukawi2](http://phs.id.au)

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details

## Acknowledgments

* https://github.com/rjeczalik/notify
