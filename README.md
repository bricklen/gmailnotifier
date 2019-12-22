## Installation of gmailnotifier plugin
1.  Install the most recent version of Golang.\
Using [Homebrew](https://brew.sh/) 
    ```
    brew update
    brew install golang
    ```
1.  Get the most recent release of BitBar. Currently that is https://github.com/matryer/bitbar/releases/tag/v1.9.2
1.  Extract the `BitBar-v1.9.2.zip` zip file (in other words, double-click to open it).
1.  Copy the BitBar.app file to the **Applications** directory.
1.  Double-click the `BitBar.app` application to open it, then click Open to install it.\
    You may need to go in to the Mac security settings to allow installation of a third party app on your Mac if your settings prevent you from installing BitBar.
1.  When prompted by BitBar, set the `plugins` directory location, this is the location where your toolbar apps will run from.\
    **NOTE**: If you only want to run this app (and no other BitBar plugins), using the `gmailnotifier/plugins` directory is fine. However, if you have other plugins, you probably want a plugins directory in a more central location (eg. `$HOME/bitbar/plugins/`.\
    If you use an external plugins directory, you will need to copy the `gmail-checker.*.cgo` executable to that plugins directory.
1.  Create a file called `.creds_gmail` in the same location as the executable (`gmailnotifier/plugins/`), then add your Gmail username and password separated by a pipe (`|`). Do this for each email account you want to check.
    ```
    examplename|mysecretpassword
    myotheremail|supersecretpass
    mygsuiteemail|myuncrackablepasswd
    ```
1.  Click the BitBar icon in your toolbar, select Preferences, then Refresh All.
1.  Your plugin should now a small mail icon with a number to the right of it

## Rebuilding the Golang executable
If you make changes to main.go, you will need to rebuild the .cgo file.
```
# From the gmailnotifier directory, build the executable:
go build -o plugins/gmail-checker.30s.cgo src/main.go
```

### Other options for the plugin
There are tutorials at https://github.com/matryer/bitbar-plugins/tree/master/Tutorial
The official guide is at https://github.com/matryer/bitbar#writing-plugins

### Changing the execution intervals
The gmail check frequency is defined by the interval in the file name, between the name and extension. For example, to check every 30 seconds, the file name would be `gmail-checker.30s.cgo`

#### Gmail App Passwords are preferred over your main email password
See https://support.google.com/accounts/answer/185833?hl=en

Excerpt
> When you use 2-Step Verification, some apps or devices may be blocked from accessing your Google Account. App Passwords are a way to let the blocked app or device access your Google Account.
   
   
   