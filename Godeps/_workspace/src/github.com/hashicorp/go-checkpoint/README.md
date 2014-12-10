# Go Checkpoint Client

[Checkpoint](http://checkpoint.hashicorp.com) is an internal service at
Hashicorp that we use to check version information, broadcoast security
bulletins, etc.

We understand that software making remote calls over the internet
for any reason can be undesirable. Because of this, Checkpoint can be
disabled in all of our software that includes it. You can view the source
of this client to see that we're not sending any private information.

**Note:** This repository is probably useless outside of internal HashiCorp
use. It is open source for disclosure and because our open source projects
must be able to link to it.
