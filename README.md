# Project Title

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

Configuration is stored in `/etc/fscanary.ini`. The configuration has a global
section for general configuration, then one or more sections to set up watches.

Example configuration file:

```
email = admin@example.com

[executables]
path = /home
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
notify = yes
quarantine = yes
dest = /quarantined
```

## Contributing

Pull requests welcomed! Please keep the following points in mind:

* One feature change/bug fix per pull request.
  * Multiple minor changes such as spelling may be in a single PR.
* Documentation must be updated with the PR where the change warrants
  documentation changes.

## Authors

* **Phillip Smith** - [fukawi2](https://phs.id.au)

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details

## Acknowledgments

* https://github.com/rjeczalik/notify
