package setup

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// osascript wrappers. We don't pass user input through the shell — all
// values go via separate process arguments so no escaping issues exist.
//
// We do, however, have to assemble a small AppleScript program as a
// string. To avoid injection inside that script, the prompt/title strings
// are passed in via osascript's positional arguments (-e ... -- arg1 arg2)
// and referenced via `item N of argv`.

// ErrDialogCancelled is returned when the user cancels a dialog.
var ErrDialogCancelled = errors.New("dialog cancelled")

// AskString shows a one-line text input dialog and returns the entered text.
func AskString(title, prompt, defaultText string) (string, error) {
	const script = `
on run argv
    set theTitle to item 1 of argv
    set thePrompt to item 2 of argv
    set theDefault to item 3 of argv
    try
        set theResponse to display dialog thePrompt with title theTitle default answer theDefault buttons {"Cancel", "OK"} default button "OK"
        return text returned of theResponse
    on error number -128
        return "__CANCELLED__"
    end try
end run
`
	out, err := runOSA(script, title, prompt, defaultText)
	if err != nil {
		return "", err
	}
	if out == "__CANCELLED__" {
		return "", ErrDialogCancelled
	}
	return out, nil
}

// AskPassword shows a hidden-answer text input. The password never crosses
// the shell — osascript passes it back to us on stdout, where we read it
// from a []byte and immediately convert to string without logging it.
func AskPassword(title, prompt string) (string, error) {
	const script = `
on run argv
    set theTitle to item 1 of argv
    set thePrompt to item 2 of argv
    try
        set theResponse to display dialog thePrompt with title theTitle default answer "" buttons {"Cancel", "OK"} default button "OK" with hidden answer
        return text returned of theResponse
    on error number -128
        return "__CANCELLED__"
    end try
end run
`
	out, err := runOSA(script, title, prompt)
	if err != nil {
		return "", err
	}
	if out == "__CANCELLED__" {
		return "", ErrDialogCancelled
	}
	return out, nil
}

// Choose shows a single-select list dialog and returns the chosen item.
func Choose(title, prompt string, items []string) (string, error) {
	if len(items) == 0 {
		return "", errors.New("Choose: no items provided")
	}
	// Build the choose-from-list call dynamically — note the items list
	// is built from `argv` items 3..N so no escaping issues remain.
	const script = `
on run argv
    set theTitle to item 1 of argv
    set thePrompt to item 2 of argv
    set theItems to {}
    repeat with i from 3 to (count of argv)
        set end of theItems to item i of argv
    end repeat
    set theChoice to choose from list theItems with title theTitle with prompt thePrompt default items {item 1 of theItems}
    if theChoice is false then return "__CANCELLED__"
    return item 1 of theChoice
end run
`
	args := append([]string{title, prompt}, items...)
	out, err := runOSA(script, args...)
	if err != nil {
		return "", err
	}
	if out == "__CANCELLED__" {
		return "", ErrDialogCancelled
	}
	return out, nil
}

// Confirm shows a yes/no dialog.
func Confirm(title, prompt string) (bool, error) {
	const script = `
on run argv
    set theTitle to item 1 of argv
    set thePrompt to item 2 of argv
    try
        display dialog thePrompt with title theTitle buttons {"No", "Yes"} default button "Yes"
        if button returned of result is "Yes" then
            return "yes"
        else
            return "no"
        end if
    on error number -128
        return "__CANCELLED__"
    end try
end run
`
	out, err := runOSA(script, title, prompt)
	if err != nil {
		return false, err
	}
	if out == "__CANCELLED__" {
		return false, ErrDialogCancelled
	}
	return out == "yes", nil
}

// Notify shows a non-blocking informational dialog with an OK button.
func Notify(title, prompt string) error {
	const script = `
on run argv
    set theTitle to item 1 of argv
    set thePrompt to item 2 of argv
    display dialog thePrompt with title theTitle buttons {"OK"} default button "OK"
    return "ok"
end run
`
	_, err := runOSA(script, title, prompt)
	return err
}

func runOSA(script string, args ...string) (string, error) {
	cmdArgs := []string{"-e", script, "--"}
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.Command("/usr/bin/osascript", cmdArgs...)
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return "", fmt.Errorf("osascript (exit %d): %s",
				ee.ExitCode(), strings.TrimSpace(string(ee.Stderr)))
		}
		return "", fmt.Errorf("osascript: %w", err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}
