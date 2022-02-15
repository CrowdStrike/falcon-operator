![CrowdStrike Falcon](https://raw.githubusercontent.com/CrowdStrike/falconpy/main/docs/asset/cs-logo.png) [![Twitter URL](https://img.shields.io/twitter/url?label=Follow%20%40CrowdStrike&style=social&url=https%3A%2F%2Ftwitter.com%2FCrowdStrike)](https://twitter.com/CrowdStrike)<br/>

# Falcon Operator
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/falcon-operator)](https://artifacthub.io/packages/search?repo=falcon-operator)
[![CI Golang Build](https://github.com/CrowdStrike/falcon-operator/actions/workflows/go.yml/badge.svg)](https://github.com/CrowdStrike/falcon-operator/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/crowdstrike/falcon-operator)](https://goreportcard.com/report/github.com/crowdstrike/falcon-operator)
[![gosec](https://github.com/CrowdStrike/falcon-operator/actions/workflows/gosec.yml/badge.svg)](https://github.com/CrowdStrike/falcon-operator/actions/workflows/gosec.yml)
[![CodeQL](https://github.com/CrowdStrike/falcon-operator/actions/workflows/codeql.yml/badge.svg)](https://github.com/CrowdStrike/falcon-operator/actions/workflows/codeql.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/crowdstrike/falcon-operator.svg)](https://pkg.go.dev/github.com/crowdstrike/falcon-operator)
[![CI Container Build](https://github.com/CrowdStrike/falcon-operator/actions/workflows/container_build.yml/badge.svg)](https://github.com/CrowdStrike/falcon-operator/actions/workflows/container_build.yml)
[![Docker Repository on Quay](https://quay.io/repository/crowdstrike/falcon-operator/status "Docker Repository on Quay")](https://quay.io/repository/crowdstrike/falcon-operator)
[![Docker Repository on Quay](https://quay.io/repository/crowdstrike/falcon-operator-bundle/status "Docker Repository on Quay")](https://quay.io/repository/crowdstrike/falcon-operator-bundle)

Falcon Operator installs CrowdStrike Falcon Container Sensor or CrowdStrike Falcon Node Sensor on the cluster.

Falcon Operator is an open source project, not a CrowdStrike product. As such it carries no formal support, expressed or implied.

## About Falcon Operator
Falcon Operator deploys CrowdStrike Falcon Workload Protection to the cluster. The operator exposes 2 custom resources that allows you to deploy either Falcon Container Sensor or Falcon Node Sensor.

## About Custom Resources

| Custom Resource                   | Description                                                      |
| :--------                         | :------------                                                    |
| [FalconContainer](docs/container) | Manages installation of Falcon Container Sensor on the cluster   |
| [FalconNodeSensor](docs/node)     | Manages installation of Falcon Linux Sensor on the cluster nodes |

Additional information can be found in [FAQ document](docs/faq.md)

## Installation Steps

Installation steps differ based on Operator Life-cycle Manager (OLM) availability. You can determine whether your cluster is using OLM by running: `kubectl get crd catalogsources.operators.coreos.com`

 - (option 1): In case your cluster **is not** using OLM (Operator Life-cycle Manager) run:
   ```
   kubectl apply -f https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/deploy/falcon-operator.yaml
   ```

 - (option 2): In case your cluster **is** using OLM run:
   ```
   OPERATOR_NAMESPACE=falcon-operator
   kubectl create ns $OPERATOR_NAMESPACE --dry-run=client -o yaml | kubectl apply -f -
   operator-sdk run bundle quay.io/crowdstrike/falcon-operator-bundle:latest --namespace $OPERATOR_NAMESPACE
   ```

After the installation concludes please proceed with deploying either [Falcon Container Sensor](docs/container) or [Falcon Node Sensor](docs/node).

## Uninstall Steps

 - To uninstall Falcon Operator run (when installed using OLM)
   ```
   operator-sdk cleanup falcon-operator --namespace falcon-operator
   ```
 - To uninstall Falcon Operator run (when installed manually)
   ```
   kubectl delete -f https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/deploy/falcon-operator.yaml
   ```

## Getting Help
If you encounter any issues while using Falcon Operator, you can create an issue on our [Github repo](https://github.com/CrowdStrike/falcon-operator) for bugs, enhancements, or other requests.

## Contributing
You can contribute by:

* Raising any issues you find using Falcon Operator
* Fixing issues by opening [Pull Requests](https://github.com/CrowdStrike/falcon-operator/pulls)
* Submitting a patch or opening a PR
* Improving documentation
* Talking about Falcon Operator

All bugs, tasks or enhancements are tracked as [GitHub issues](https://github.com/CrowdStrike/falcon-operator/issues).

## Additional Resources
 - CrowdStrike Container Security: [Product Page](https://www.crowdstrike.com/products/cloud-security/falcon-cloud-workload-protection/container-security/)
 - So You Think Your Containers Are Secure? Four Steps to Ensure a Secure Container Deployment: [Blog Post](https://www.crowdstrike.com/blog/four-steps-to-ensure-a-secure-containter-deployment/)
 - Container Security With CrowdStrike: [Blog Post](https://www.crowdstrike.com/blog/tech-center/container-security/)
 - To learn more about Falcon Container Sensor for Linux: [Deployment Guide](https://falcon.crowdstrike.com/support/documentation/146/falcon-container-sensor-for-linux), [Release Notes](https://falcon.crowdstrike.com/support/news/release-notes-falcon-container-sensor-for-linux)
 - [Developer Documentation](docs/developer_guide.md)
