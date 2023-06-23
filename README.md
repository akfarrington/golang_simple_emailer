### golang_simple_emailer

A simple bulk emailer written in go. Basically put CSVs, Gomail, and Golang text templates together.

Basically, my wife needed to send lots of emails so I made this. It was written quickly, mostly like a script. It's not a "good" program. There are things that I would want to improve, especially if my wife were to actually use it. I ended up teaching her how to use mail merge because her company doesn't permit app passwords to log in to their work emails. So, making this was kind of mostly a waste of time, but now I have a new thing to put on Github. Yay!

If I were to improve this thing, I'd do some of the following:
- print a pretty help message when flags are missed or wrong or when `-help` is typed
- move optional cc emails to the commandline options, rather than using the .env file
- general improvements - the code is kind of a huge mess - all saved in main.go
- figure out how to work with csv files more dynamically - right now all the program can do is accept csv files with a `"name","email"` format
- add a command line option to use plain emails instead of html. That or automatically detect based on which kind of file is saved there.
- handle errors more gracefully instead of panicking anytime something weird happens

# How to use:

1. Edit the `emailer.env` file with your email info. Keep in mind that email services that use MFA won't work. You'd need to get an app password for this executable to work.
2. Edit the `email.html` file. Use `{{.Name}}` (case sensitive) where you want the program to fill in the name.
3. Assuming the executable is named `emailer`, you would run `emailer -subject="email subject"` first.
4. Check the test output in the folder `test-emails` to verify everything looks good.
5. Run the executable with the `-run` flag, i.e. `emailer -subject="email" -run`.
6. Profit