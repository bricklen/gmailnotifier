# Prerequisites
* [BitBar](https://github.com/matryer/bitbar)
* Recent version of Golang

# tl;dr installation instructions
1.  Install BitBar.
1.  Copy the `gmailnotifier.*.cgo` executable from the `plugins` directory of this github repo to a local directory on your Mac.
1.  Create the `.creds_gmail` file in the same local directory, this file stores the `email|password` credentials.

# Detailed installation instructions 

### BitBar installation
1.  Get the most recent release of BitBar from https://github.com/matryer/bitbar/releases/
1.  Extract/open the `BitBar-X.Y.Z.zip` zip file. `X.Y.Z` here is the package version.
1.  Copy the `BitBar.app` file to the **Applications** directory.
1.  Launch the `BitBar` app. For example, using Spotlight (Command + Space), type in BitBar and you should see the application listed.
    Launch it, then click Open to install it.
    
    You may need to go to *System Preferences* -> *Security & Privacy* -> *General* and allow the app to be opened. This step applies to installing `gmailnotifier` too, because these are third party apps, not ones from the Mac app store.
1.  You will be prompted to choose a location where the plugins are located. For now, you can cancel out of this, we'll set that location once `gmailnotifier` is installed.

### gmailnotifier installation
1.  Download the most recent release from https://github.com/bricklen/gmailnotifier/releases
1.  Extract the zip file (or tarball). If you're unsure where to extract to, create a directory under $HOME called `apps` and extract the zip file there.
1.  Launch the BitBar app if you haven't already.
1.  Click on the BitBar text/icon in the toolbar, select *Change Plugin Folder*, then set the plugins directory location to where your toolbar apps will run from. This should be `gmailnotifier-X.Y/plugins`, where `X.Y` is the version. (see NOTE below for alternatives).
    
    You can change this plugins setting at any time by clicking on your BitBar toolbar app, then clicking *Preferences* ->  *Change Plugin Folder*.
    
    **NOTE**: If you only want to run this app (and no other BitBar plugins), using the `gmailnotifier-X.Y/plugins` directory is fine. However, if you have other plugins, you probably want a plugins directory in a more central location (eg. `$HOME/bitbar/plugins/`.
    If you do use an external BitBar plugins directory, you will need to copy the `gmailnotifier.*.cgo` executable to that plugins directory.
1.  If you get a warning like "*gmailnotifier.30s.cgo cannot be opened because the developer cannot be verified*", you will need go to *System Preferences* -> *Security & Privacy* -> *General* and allow the app to be opened.
1.  If you needed to grant permission to install the gmailnotifier app, repeat the earlier BitBar *Change Plugin Folder* step (step #4)
1.  Once gmailnotifier is successfully installed, add a file called `.creds_gmail` to same location as the executable (eg. `gmailnotifier-X.Y/plugins/`). This file must exist for the plugin to know which accounts to check.
    
    Add the Gmail username and password to .creds_gmail separated by a pipe (`|`).
    
    Do this for each email account you want to check, one *username|password* pair per line. 

    #### Example
    ```
    examplename|mysecretpassword
    myotheremail|supersecretpass
    my-gsuite-email@acme.com|myuncrackablepasswd
    ```
1.  Click the BitBar icon in your toolbar, select *Preferences* -> *Refresh All*.
1.  Your plugin should now show as a small mail icon with the number of unread emails to the right of it.

### Note about G-Suite (non-gmail) domains
This plugin works for those, but you will need to supply your full organization address as your username, whereas if it is for a regular Gmail account, you only need to supply your username.

### Directory layout
```
gmailnotifier-X.Y
├── .gitignore
├── LICENSE
├── README.md
├── plugins
│   ├── .creds_gmail
│   └── gmailnotifier.30s.cgo
└── src
    └── main.go
```

## Rebuilding the Golang executable
If you make changes to `main.go`, you will need to rebuild the `.cgo` executable file.

From the `gmailnotifier-X.Y` directory, build the executable:
```
go build -o plugins/gmailnotifier.30s.cgo src/main.go
```

### Other options for the plugin
* The official guide is at https://github.com/matryer/bitbar#writing-plugins
* Community tutorials at https://github.com/matryer/bitbar-plugins/tree/master/Tutorial

### Changing the execution intervals
The notifier check frequency is defined by the interval in the file name (between the name and extension). For example, to check every 30 seconds, the file name would be `gmailnotifier.30s.cgo`

### About the credentials file
If you fork or build from source, **be sure** to maintain the credentials file name of `.creds_gmail`, or if you change it, make sure to update the `.gitignore` file with the new creds file name so that it does not get inadvertently committed to your github repo. If you end up exposing usernames and passwords in a github repo you're going to be in for a bad time.

#### Use Gmail App Passwords instead of your login password
I recommend using App Passwords as they make it easy to have a separate password for each application that needs to connect to your gmail account, and you can revoke them individually if required.
See https://support.google.com/accounts/answer/185833?hl=en

Excerpt
> When you use 2-Step Verification, some apps or devices may be blocked from accessing your Google Account. App Passwords are a way to let the blocked app or device access your Google Account.

## Privacy notice
This little app does not save, nor send your credentials (or anything else for that matter) anywhere, except directly to https://mail.google.com/mail to get the unread mail counts.

You are encouraged to review the `main.go` source file - and to build a new executable from that - if you have any concerns about the code.

## Disclaimer
See the LICENSE about disclaimers of liability. As with all Open Source software: use at your own risk. 

## TODO
- Investigate rewriting this script to use the Google native API tools, https://godoc.org/google.golang.org/api/gmail/
- Add option to delete a message. Prompt to confirm.

