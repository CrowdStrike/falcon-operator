# Advanced Settings

 Some of the operator's configurable settings involve features that conflict with established industry norms. These options are disabled by default as they introduce various amounts of risk. While their use is not recommended they can be enabled in the `advanced` section of each resource spec. What follows is a brief overview of the issues surrounding these settings.

## The Golden Rule of Kubernetes

A fundamental principle underlying all Kubernetes operation is repeatability. Any given configuration should always produce the same result regardless of when or where it is applied or by whom. Another way of saying this is that a cluster should only ever do something because somebody explicitly called for it to happen. Anything that has variable behavior introduces uncertainty into the environment, and this can lead to problems that are difficult to diagnose.

A common example is the use of image tags. These operate like pointers with many of the same concerns. The image they refer to can change without warning, and that can cause trouble.

Consider a container spec that uses `nginx:latest`. What exactly will this deploy? Some version of nginx, presumably, but which version? What if it's not the version expected by the rest of the system? What if it's incompatible with other things in the cluster? Maybe everything works fine today, but what if tomorrow the container is moved to a different node? This tears down the old one and launches a new one. What if `latest` has changed to something new that breaks everything? There's no way to detect this beforehand.

It is for these reasons and others that such practices are discouraged. A better approach given the above scenario is to use explicit image hashes. Instead of `nginx:latest`, one could use `nginx@sha256:447a8665...`. This uniquely identifies a particular version and package of nginx. It will never be anything else. All of the questions raised above become irrelevant. It is known what version will be deployed. It is known it will be the expected version. It is known new containers won't use anything else. It is safe.

## Falcon's Advanced Options

Only some of the resources provided by the operator have advanced properties. Each keeps them in slightly different places:

* `spec.advanced` for FalconContainer
* `spec.node.advanced` for FalconNodeSensor

Any options that go against recommended practices can be found here. Presently, that includes settings that affect the selection of Falcon sensor versions, which brings all of the issues of image tags described above. Details on these settings can be found in the respective resource documents.

## More Information

The issues around these advanced settings can be quite involved. The following are other resources that go into greater depth:

* [Attack of the Mutant Tags! Or Why Tag Mutability is a Real Security Threat](https://sysdig.com/blog/toctou-tag-mutability/)
* [How to Ensure Consistent Kubernetes Container Versions](https://www.gremlin.com/blog/kubernetes-container-image-version-uniformity)
