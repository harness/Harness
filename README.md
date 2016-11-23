[![Build Status](http://beta.drone.io/api/badges/drone/drone/status.svg)](http://beta.drone.io/drone/drone)
![Release Status](https://img.shields.io/badge/status-beta-yellow.svg?style=flat)
[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/drone/drone?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

Drone is a Continuous Integration platform built on container technology. Every build is executed inside an ephemeral Docker container, giving developers complete control over their build environment with guaranteed isolation.

### Goals

Drone's prime directive is to help teams [ship code like GitHub](https://github.com/blog/1241-deploying-at-github#always-be-shipping). Drone is easy to install, setup and maintain and offers a powerful container-based plugin system. Drone aspires to be an industry-wide replacement for Jenkins.

### Documentation

Drone documentation is organized into several categories:

* [Setup Guide](http://readme.drone.io/setup/overview)
* [Build Guide](http://readme.drone.io/usage/overview)
* [Plugin Guide](http://readme.drone.io/devs/plugins)
* [CLI Reference](http://readme.drone.io/devs/cli/)
* [API Reference](http://readme.drone.io/devs/api/builds)

### Documentation for 0.5 (unstable)

If you are using the 0.5 unstable release (master branch) please see the updated [documentation](http://readme.drone.io/0.5).



### Community, Help

Contributions, questions, and comments are welcomed and encouraged. Drone developers hang out in the [drone/drone](https://gitter.im/drone/drone) room on gitter. We ask that you please post your questions to [gitter](https://gitter.im/drone/drone) before creating an issue.

### Installation

Please see our [installation guide](http://readme.drone.io/setup/overview) to install the official Docker image.

### From Source

Build from source (artifacts will be in `./release`):

```sh
git clone git://github.com/drone/drone.git
cd drone
make docker-build
```

If you are having trouble building this project please reference its `.drone.yml` file. Everything you need to know about building Drone is defined in that file.
