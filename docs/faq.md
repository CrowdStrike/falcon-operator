# FAQ - Frequently Asked Questions

### What network connections the operator makes outside the cluster?

 - Operator image is available in quay.io (the image itself can be mirrored)
 - Operator will reach to your particular Falcon cloud region and to `registry.crowdstrike.com`
 - If Falcon Cloud is set to autodiscover, the operator may reach also to Falcon Cloud Region **us-1**
