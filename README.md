# Cabourotte

Cabourotte is a tool that can be configure to execute health checks on your infrastructure.

It's used to execute health checks on the [Appclacks Cloud platform](https://appclacks.com/). It's a free software that you can also install on your private infrastructure.

The Appclacks Cloud platform documentation is available at [https://doc.appclacks.com/](https://doc.appclacks.com/) and the Cabourotte documentation is available at [https://cabourotte.appclacks.com/](https://cabourotte.appclacks.com/)

---

IT infrastructures are complex. We have more and more equipments, machines, and services to manage. Infrastructures are also more dynamic, with services which can be scaled up and down depending on usage.

The rise of containers orchestrators also made networking more complex. On a network failure, a service could be reachable from one part of your infrastructure but not from another one.

Cabourotte is a tool which allow you to execute healthchecks (HTTP(s), TCP, DNS, TLS including certificate expiration notice, arbitrary commands) on your infrastructure. It already supports various features including:

- Configurable by using a YAML file, or by using the API. Using the API allows you to dynamically add, update, or remove healthchecks definitions. The API also allows you to list configured healthchecks and to get the latest status for each healthcheck.
- HTTP service discovery: You can easily integration Cabourotte with anything you want.
- Prometheus integration: the healthchecks results and executions time are exposed on a Prometheus endpoint alongside various internal metrics.
- Support exporters, which can be configured to push the healthchecks results to another systems.
- `One-Off` healthchecks: You can send requests to the API to execute arbitrary healthchecks and get the healthchecks results in the responses.
- Hot reload on a SIGHUP.
- A small frontend to see the current healthchecks status

Lightweight, written in Golang, Cabourotte can run everywhere to detect services and network failures.
