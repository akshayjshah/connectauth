Contributing
============

I'd appreciate your help making `connectauth` better! Please keep in mind that
this is a personal project - while I'm the primary author of `connect-go` and
the Connect protocol, I can't devote quite as much time to this project.
Especially if your change adds new exported APIs, please open an issue to
discuss the change before making a PR. :heart_eyes_cat:

Most importantly, please remember to treat your fellow contributors with
respect!

## Build and test

`connectauth` is simple: you can build and test with the usual `go test .`, or you
can use the Makefile to match CI more closely (`make help` lists the available
targets). If you're opening a PR, you're most likely to get my attention if
you:

* Add tests to cover your changes.
* Write a [good commit message][commit-message].
* Maintain backward compatibility.
* Stay patient. This isn't my day job. :wink:

[commit-message]: http://tbaggery.com/2008/04/19/a-note-about-git-commit-messages.html
