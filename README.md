## Prerequisites
Install the most recent version of Golang, if it is not already installed.
Using [Homebrew](https://brew.sh/) 
```
brew update
brew install golang
```

## BitBar installation
1.  Get the most recent release of BitBar from https://github.com/matryer/bitbar/releases/
1.  Extract the `BitBar-X.Y.Z.zip` zip file (eg. double-click to open it). `X.Y.Z` here is the package version.
1.  Copy the `BitBar.app` file to the **Applications** directory.
1.  Launch the `BitBar` app. For example, using Spotlight (Command + Space), type in BitBar and you should see the application listed.
    Double click to launch it, then click Open to install it.
    
    You may need to go to *System Preferences* -> *Security & Privacy* -> *General* and allow the app to be opened. This step applies to installing `gmailnotifier` too, because these are third party apps, not ones from the Mac app store.
1.  You will be prompted to choose a location where the plugins are located. For now, you can cancel out of this, we'll set that location once `gmailnotifier` is installed.

## gmailnotifier installation
1.  Download the most recent release from https://github.com/bricklen/gmailnotifier/releases
1.  Extract the zip file (or tarball). If you're unsure where to extract to, create a directory under $HOME called `apps` and extract the zip file there.
1.  Launch the BitBar app if you haven't already.
1.  Click on the BitBar text/icon in the toolbar, select *Change Plugin Folder*, then set the plugins directory location to where your toolbar apps will run from. This should be `gmailnotifier-X.Y/plugins`, where `X.Y` is the version. (see NOTE below for alternatives).
    
    You can change this plugins setting at any time by clicking on your BitBar toolbar app, then clicking *Preferences* ->  *Change Plugin Folder*.
    
    **NOTE**: If you only want to run this app (and no other BitBar plugins), using the `gmailnotifier-X.Y/plugins` directory is fine. However, if you have other plugins, you probably want a plugins directory in a more central location (eg. `$HOME/bitbar/plugins/`.
    If you do use an external BitBar plugins directory, you will need to copy the `gmailnotifier.*.cgo` executable to that plugins directory.
1.  If you get a warning like "*gmailnotifier.30s.cgo cannot be opened because the developer cannot be verified*", you will need go to *System Preferences* -> *Security & Privacy* -> *General* and allow the app to be opened.
1.  Once gmailnotifier is successfully installed, modify the `.creds_gmail` file, which should be in the same location as the executable (eg. `gmailnotifier-X.Y/plugins/`). If it doesn't exist, create it.
    
    Add your Gmail username and password separated by a pipe (`|`).
    
    Do this for each email account you want to check, one *username|password* pair per line. 

    #### Example
    ```
    examplename|mysecretpassword
    myotheremail|supersecretpass
    someothergmail|myuncrackablepasswd
    ```
1.  Click the BitBar icon in your toolbar, select *Preferences* -> *Refresh All*.
1.  Your plugin should now show as a small mail icon with the number of unread emails to the right of it.

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

#### Gmail App Passwords are preferred over your main email password
See https://support.google.com/accounts/answer/185833?hl=en

Excerpt
> When you use 2-Step Verification, some apps or devices may be blocked from accessing your Google Account. App Passwords are a way to let the blocked app or device access your Google Account.
   
   
   