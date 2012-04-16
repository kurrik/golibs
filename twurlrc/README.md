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

If you are on Lion and  have issues with Bazaar, e.g:

    bzr: ERROR: Couldn't import bzrlib and dependencies.

Then run:

    sudo sed -i '' s,/usr/bin/python,/usr/bin/python2.6, /usr/local/bin/bzr

Using
-----
Import `github.com/kurrik/twurlrc` in your code.

See `twurlrc_test.go` for usage.
