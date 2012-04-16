A library for reading the Twitter configuration files written by Twurlrc.
See https://github.com/marcel/twurl for more information.

Dependencies
------------
You'll need a working copy of Bazaar to pull the launchpad.net dependency.
Bazaar can be obtained here: http://wiki.bazaar.canonical.com/

Installing
----------
Run:

    go get -fix github.com/kurrik/twurlrc

Using
-----
Import `github.com/kurrik/twurlrc` in your code.

See `twurlrc_test.go` for usage, but generally you should be able to do:

    conf, err := twurlrc.Load(twurlrc.GetDefaultPath())
    cred := conf.GetDefaultCredentials()

